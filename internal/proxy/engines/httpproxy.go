/**
* Copyright 2018 Comcast Cable Communications Management, LLC
* Licensed under the Apache License, Version 2.0 (the "License");
* you may not use this file except in compliance with the License.
* You may obtain a copy of the License at
* http://www.apache.org/licenses/LICENSE-2.0
* Unless required by applicable law or agreed to in writing, software
* distributed under the License is distributed on an "AS IS" BASIS,
* WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
* See the License for the specific language governing permissions and
* limitations under the License.
 */

package engines

import (
	"bytes"
	"io"
	"io/ioutil"
	"math"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Comcast/trickster/internal/cache/status"
	"github.com/Comcast/trickster/internal/config"
	"github.com/Comcast/trickster/internal/proxy/headers"
	"github.com/Comcast/trickster/internal/proxy/params"
	"github.com/Comcast/trickster/internal/proxy/request"
	"github.com/Comcast/trickster/internal/timeseries"
	"github.com/Comcast/trickster/internal/util/log"
	"github.com/Comcast/trickster/internal/util/metrics"
)

// Reqs is for Progressive Collapsed Forwarding
var Reqs sync.Map

// HTTPBlockSize represents 32K of bytes
const HTTPBlockSize = 32 * 1024

// DoProxy proxies an inbound request to its corresponding upstream origin with no caching features
func DoProxy(w io.Writer, r *http.Request) *http.Response {

	rsc := request.GetResources(r)
	pc := rsc.PathConfig
	oc := rsc.OriginConfig

	start := time.Now()
	var elapsed time.Duration
	var cacheStatusCode status.LookupStatus
	var resp *http.Response
	var reader io.Reader
	if pc == nil || pc.CollapsedForwardingType != config.CFTypeProgressive {
		reader, resp, _ = PrepareFetchReader(r)
		cacheStatusCode = setStatusHeader(resp.StatusCode, resp.Header)
		writer := PrepareResponseWriter(w, resp.StatusCode, resp.Header)
		if writer != nil && reader != nil {
			io.Copy(writer, reader)
		}
	} else {
		pr := newProxyRequest(r, w)
		key := oc.CacheKeyPrefix + "." + pr.DeriveCacheKey(nil, "")
		result, ok := Reqs.Load(key)
		if !ok {
			var contentLength int64
			reader, resp, contentLength = PrepareFetchReader(r)
			cacheStatusCode = setStatusHeader(resp.StatusCode, resp.Header)
			writer := PrepareResponseWriter(w, resp.StatusCode, resp.Header)
			// Check if we know the content length and if it is less than our max object size.
			if contentLength != 0 && contentLength < int64(oc.MaxObjectSizeBytes) {
				pcf := NewPCF(resp, contentLength)
				Reqs.Store(key, pcf)
				// Blocks until server completes
				go func() {
					io.Copy(pcf, reader)
					pcf.Close()
					Reqs.Delete(key)
				}()
				pcf.AddClient(writer)
			}
		} else {
			pcf, _ := result.(ProgressiveCollapseForwarder)
			resp = pcf.GetResp()
			writer := PrepareResponseWriter(w, resp.StatusCode, resp.Header)
			pcf.AddClient(writer)
		}
	}
	elapsed = time.Since(start)
	recordResults(r, "HTTPProxy", cacheStatusCode, resp.StatusCode, r.URL.Path, "", elapsed.Seconds(), nil, resp.Header)
	return resp
}

// PrepareResponseWriter prepares a response and returns an io.Writer for the data to be written to.
// Used in Respond.
func PrepareResponseWriter(w io.Writer, code int, header http.Header) io.Writer {
	if rw, ok := w.(http.ResponseWriter); ok {
		h := rw.Header()
		headers.Merge(h, header)
		headers.AddResponseHeaders(h)
		rw.WriteHeader(code)
		return rw
	}
	return w
}

// PrepareFetchReader prepares an http response and returns io.ReadCloser to
// provide the response data, the response object and the content length.
// Used in Fetch.
func PrepareFetchReader(r *http.Request) (io.ReadCloser, *http.Response, int64) {

	rsc := request.GetResources(r)
	pc := rsc.PathConfig
	oc := rsc.OriginConfig

	var rc io.ReadCloser

	if r != nil && r.Header != nil {
		headers.AddProxyHeaders(r.RemoteAddr, r.Header)
	}

	headers.RemoveClientHeaders(r.Header)

	if pc != nil {
		headers.UpdateHeaders(r.Header, pc.RequestHeaders)
		params.UpdateParams(r.URL.Query(), pc.RequestParams)
	}

	r.RequestURI = ""
	resp, err := oc.HTTPClient.Do(r)
	if err != nil {
		log.Error("error downloading url", log.Pairs{"url": r.URL.String(), "detail": err.Error()})
		// if there is an err and the response is nil, the server could not be reached; make a 502 for the downstream response
		if resp == nil {
			resp = &http.Response{StatusCode: http.StatusBadGateway, Request: r, Header: make(http.Header)}
		}
		if pc != nil {
			headers.UpdateHeaders(resp.Header, pc.ResponseHeaders)
		}
		return nil, resp, 0
	}

	originalLen := int64(-1)
	if v, ok := resp.Header[headers.NameContentLength]; ok {
		originalLen, err = strconv.ParseInt(strings.Join(v, ""), 10, 64)
		if err != nil {
			originalLen = -1
		}
		resp.ContentLength = int64(originalLen)
	}
	rc = resp.Body

	// warn if the clock between trickster and the origin is off by more than 1 minute
	if date := resp.Header.Get(headers.NameDate); date != "" {
		d, err := http.ParseTime(date)
		if err == nil {
			if offset := time.Since(d); time.Duration(math.Abs(float64(offset))) > time.Minute {
				log.WarnOnce("clockoffset."+oc.Name,
					"clock offset between trickster host and origin is high and may cause data anomalies",
					log.Pairs{
						"originName":    oc.Name,
						"tricksterTime": strconv.FormatInt(d.Add(offset).Unix(), 10),
						"originTime":    strconv.FormatInt(d.Unix(), 10),
						"offset":        strconv.FormatInt(int64(offset.Seconds()), 10) + "s",
					})
			}
		}
	}

	hasCustomResponseBody := false
	resp.Header.Del(headers.NameContentLength)

	if pc != nil {
		headers.UpdateHeaders(resp.Header, pc.ResponseHeaders)
		hasCustomResponseBody = pc.HasCustomResponseBody
	}

	if hasCustomResponseBody {
		rc = ioutil.NopCloser(bytes.NewBuffer(pc.ResponseBodyBytes))
	}

	return rc, resp, originalLen
}

// Respond sends an HTTP Response down to the requesting client
func Respond(w http.ResponseWriter, code int, header http.Header, body []byte) {
	PrepareResponseWriter(w, code, header)
	w.Write(body)
}

func setStatusHeader(httpStatus int, header http.Header) status.LookupStatus {
	st := status.LookupStatusProxyOnly
	if httpStatus >= http.StatusBadRequest {
		st = status.LookupStatusProxyError
	}
	headers.SetResultsHeader(header, "HTTPProxy", st.String(), "", nil)
	return st
}

func recordResults(r *http.Request, engine string, cacheStatus status.LookupStatus, statusCode int, path, ffStatus string, elapsed float64, extents timeseries.ExtentList, header http.Header) {

	rsc := request.GetResources(r)
	pc := rsc.PathConfig
	oc := rsc.OriginConfig

	status := cacheStatus.String()

	if pc != nil && !pc.NoMetrics {
		httpStatus := strconv.Itoa(statusCode)
		metrics.ProxyRequestStatus.WithLabelValues(oc.Name, oc.OriginType, r.Method, status, httpStatus, path).Inc()
		if elapsed > 0 {
			metrics.ProxyRequestDuration.WithLabelValues(oc.Name, oc.OriginType, r.Method, status, httpStatus, path).Observe(elapsed)
		}
	}
	headers.SetResultsHeader(header, engine, status, ffStatus, extents)
}
