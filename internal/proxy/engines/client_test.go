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
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/Comcast/trickster/internal/cache"
	"github.com/Comcast/trickster/internal/config"
	"github.com/Comcast/trickster/internal/proxy/errors"
	tm "github.com/Comcast/trickster/internal/proxy/model"
	tt "github.com/Comcast/trickster/internal/proxy/timeconv"
	"github.com/Comcast/trickster/internal/routing"
	"github.com/Comcast/trickster/internal/timeseries"
	"github.com/Comcast/trickster/internal/util/md5"
	tu "github.com/Comcast/trickster/internal/util/testing"
	"github.com/Comcast/trickster/pkg/sort/times"

	"github.com/prometheus/common/model"
)

// Prometheus API
const (
	APIPath      = "/api/v1/"
	mnQueryRange = "query_range"
	mnQuery      = "query"
	mnLabels     = "label/" + model.MetricNameLabel + "/values"
	mnSeries     = "series"
	mnHealth     = "health"
)

// Origin Types
const (
	otPrometheus = "prometheus"
)

// Prometheus Response Values
const (
	rvSuccess = "success"
	rvMatrix  = "matrix"
	rvVector  = "vector"
)

// Common URL Parameter Names
const (
	upQuery   = "query"
	upStart   = "start"
	upEnd     = "end"
	upStep    = "step"
	upTimeout = "timeout"
	upTime    = "time"
)

// Client Implements Proxy Client Interface
type PromTestClient struct {
	name      string
	user      string
	pass      string
	config    *config.OriginConfig
	cache     cache.Cache
	webClient *http.Client

	fftime          time.Time
	InstantCacheKey string
	RangeCacheKey   string
}

func newPromTestClient(name string, config *config.OriginConfig, cache cache.Cache) *PromTestClient {

	return &PromTestClient{name: name, config: config, cache: cache, webClient: tu.NewTestWebClient()}
}

// Configuration returns the upstream Configuration for this Client
func (c *PromTestClient) Configuration() *config.OriginConfig {
	return c.config
}

// HTTPClient returns the HTTP Client for this origin
func (c *PromTestClient) HTTPClient() *http.Client {
	return c.webClient
}

// Name returns the name of the upstream Configuration proxied by the Client
func (c *PromTestClient) Name() string {
	return c.name
}

// Cache returns and handle to the Cache instance used by the Client
func (c *PromTestClient) Cache() cache.Cache {
	return c.cache
}

// parseTime converts a query time URL parameter to time.Time.
// Copied from https://github.com/prometheus/prometheus/blob/master/web/api/v1/api.go
func parseTime(s string) (time.Time, error) {
	if t, err := strconv.ParseFloat(s, 64); err == nil {
		s, ns := math.Modf(t)
		ns = math.Round(ns*1000) / 1000
		return time.Unix(int64(s), int64(ns*float64(time.Second))), nil
	}
	if t, err := time.Parse(time.RFC3339Nano, s); err == nil {
		return t, nil
	}
	return time.Time{}, fmt.Errorf("cannot parse %q to a valid timestamp", s)
}

// parseDuration parses prometheus step parameters, which can be float64 or durations like 1d, 5m, etc
// the proxy.ParseDuration handles the second kind, and the float64's are handled here
func parseDuration(input string) (time.Duration, error) {
	v, err := strconv.ParseFloat(input, 64)
	if err != nil {
		return tt.ParseDuration(input)
	}
	// assume v is in seconds
	return time.Duration(int64(v)) * time.Second, nil
}

// ParseTimeRangeQuery parses the key parts of a TimeRangeQuery from the inbound HTTP Request
func (c *PromTestClient) ParseTimeRangeQuery(r *tm.Request) (*timeseries.TimeRangeQuery, error) {

	trq := &timeseries.TimeRangeQuery{Extent: timeseries.Extent{}}
	qp := r.URL.Query()

	trq.Statement = qp.Get(upQuery)
	if trq.Statement == "" {
		return nil, errors.MissingURLParam(upQuery)
	}

	if p := qp.Get(upStart); p != "" {
		t, err := parseTime(p)
		if err != nil {
			return nil, err
		}
		trq.Extent.Start = t
	} else {
		return nil, errors.MissingURLParam(upStart)
	}

	if p := qp.Get(upEnd); p != "" {
		t, err := parseTime(p)
		if err != nil {
			return nil, err
		}
		trq.Extent.End = t
	} else {
		return nil, errors.MissingURLParam(upEnd)
	}

	if p := qp.Get(upStep); p != "" {
		step, err := parseDuration(p)
		if err != nil {
			return nil, err
		}
		trq.Step = step
	} else {
		return nil, errors.MissingURLParam(upStep)
	}

	if strings.Index(trq.Statement, " offset ") > -1 {
		trq.IsOffset = true
		r.FastForwardDisable = true
	}

	return trq, nil
}

// DeriveCacheKey calculates a query-specific keyname based on the prometheus query in the user request
func (c *PromTestClient) DeriveCacheKey(r *tm.Request, extra string) string {

	isInstant := strings.HasSuffix(r.URL.Path, "/query")
	isRange := strings.HasSuffix(r.URL.Path, "/query_range")

	if isInstant && c.InstantCacheKey != "" {
		return c.InstantCacheKey
	}
	if isRange && c.RangeCacheKey != "" {
		return c.RangeCacheKey
	}

	params := r.URL.Query()
	key := md5.Checksum(r.URL.Path + params.Get(upQuery) + params.Get(upStep) + params.Get(upTime) + extra)

	return key
}

// BaseURL returns a URL in the form of scheme://host/path based on the proxy configuration
func (c *PromTestClient) BaseURL() *url.URL {
	u := &url.URL{}
	u.Scheme = c.config.Scheme
	u.Host = c.config.Host
	u.Path = c.config.PathPrefix
	return u
}

// BuildUpstreamURL will merge the downstream request with the BaseURL to construct the full upstream URL
func (c *PromTestClient) BuildUpstreamURL(r *http.Request) *url.URL {
	u := c.BaseURL()

	if strings.HasPrefix(r.URL.Path, "/"+c.name+"/") {
		u.Path += strings.Replace(r.URL.Path, "/"+c.name+"/", "/", 1)
	} else {
		u.Path += r.URL.Path
	}

	u.RawQuery = r.URL.RawQuery
	u.Fragment = r.URL.Fragment
	u.User = r.URL.User
	return u
}

// SetExtent will change the upstream request query to use the provided Extent
func (c *PromTestClient) SetExtent(r *tm.Request, extent *timeseries.Extent) {
	params := r.URL.Query()
	params.Set(upStart, strconv.FormatInt(extent.Start.Unix(), 10))
	params.Set(upEnd, strconv.FormatInt(extent.End.Unix(), 10))
	r.URL.RawQuery = params.Encode()
}

// FastForwardURL returns the url to fetch the Fast Forward value based on a timerange url
func (c *PromTestClient) FastForwardURL(r *tm.Request) (*url.URL, error) {

	u := tm.CopyURL(r.URL)

	if strings.HasSuffix(u.Path, "/query_range") {
		u.Path = u.Path[0 : len(u.Path)-6]
	}

	// let the test client have a way to throw an error
	if strings.Index(u.RawQuery, "throw_ffurl_error=1") != -1 {
		return nil, fmt.Errorf("This is an intentional test error: %s", ":)")
	}

	p := u.Query()
	p.Del(upStart)
	p.Del(upEnd)
	p.Del(upStep)

	if c.fftime.IsZero() {
		c.fftime = time.Now()
	}
	p.Set("time", strconv.FormatInt(c.fftime.Unix(), 10))

	u.RawQuery = p.Encode()

	return u, nil
}

// VectorEnvelope represents a Vector response object from the Prometheus HTTP API
type VectorEnvelope struct {
	Status string     `json:"status"`
	Data   VectorData `json:"data"`
}

// VectorData represents the Data body of a Vector response object from the Prometheus HTTP API
type VectorData struct {
	ResultType string       `json:"resultType"`
	Result     model.Vector `json:"result"`
}

// MatrixEnvelope represents a Matrix response object from the Prometheus HTTP API
type MatrixEnvelope struct {
	Status       string                `json:"status"`
	Data         MatrixData            `json:"data"`
	ExtentList   timeseries.ExtentList `json:"extents,omitempty"`
	StepDuration time.Duration         `json:"step,omitempty"`

	timestamps map[time.Time]bool // tracks unique timestamps in the matrix data
	tslist     times.Times
	isSorted   bool // tracks if the matrix data is currently sorted
	isCounted  bool // tracks if timestamps slice is up-to-date
}

// MatrixData represents the Data body of a Matrix response object from the Prometheus HTTP API
type MatrixData struct {
	ResultType string       `json:"resultType"`
	Result     model.Matrix `json:"result"`
}

// MarshalTimeseries converts a Timeseries into a JSON blob
func (c *PromTestClient) MarshalTimeseries(ts timeseries.Timeseries) ([]byte, error) {
	// Marshal the Envelope back to a json object for Cache Storage
	if c.RangeCacheKey == "failkey" {
		return nil, fmt.Errorf("generic failure for testing purposes (key: %s)", c.RangeCacheKey)
	}
	return json.Marshal(ts)
}

// UnmarshalTimeseries converts a JSON blob into a Timeseries
func (c *PromTestClient) UnmarshalTimeseries(data []byte) (timeseries.Timeseries, error) {
	me := &MatrixEnvelope{}
	err := json.Unmarshal(data, &me)
	return me, err
}

// UnmarshalInstantaneous converts a JSON blob into an Instantaneous Data Point
func (c *PromTestClient) UnmarshalInstantaneous(data []byte) (timeseries.Timeseries, error) {
	ve := &VectorEnvelope{}
	err := json.Unmarshal(data, &ve)
	if err != nil {
		return nil, err
	}
	return ve.ToMatrix(), nil
}

// ToMatrix converts a VectorEnvelope to a MatrixEnvelope
func (ve *VectorEnvelope) ToMatrix() *MatrixEnvelope {
	me := &MatrixEnvelope{}
	me.Status = ve.Status
	me.Data = MatrixData{
		ResultType: "matrix",
		Result:     make(model.Matrix, 0, len(ve.Data.Result)),
	}
	var ts time.Time
	for _, v := range ve.Data.Result {
		v.Timestamp = model.TimeFromUnix(v.Timestamp.Unix()) // Round to nearest Second
		ts = v.Timestamp.Time()
		me.Data.Result = append(me.Data.Result, &model.SampleStream{Metric: v.Metric, Values: []model.SamplePair{model.SamplePair{Timestamp: v.Timestamp, Value: v.Value}}})
	}
	me.ExtentList = timeseries.ExtentList{timeseries.Extent{Start: ts, End: ts}}
	return me
}

// Times represents an array of Prometheus Times
type Times []model.Time

// Step returns the step for the Timeseries
func (me *MatrixEnvelope) Step() time.Duration {
	return me.StepDuration
}

// SetStep sets the step for the Timeseries
func (me *MatrixEnvelope) SetStep(step time.Duration) {
	me.StepDuration = step
}

// Merge merges the provided Timeseries list into the base Timeseries (in the order provided) and optionally sorts the merged Timeseries
func (me *MatrixEnvelope) Merge(sort bool, collection ...timeseries.Timeseries) {
	meMetrics := make(map[string]*model.SampleStream)
	for _, s := range me.Data.Result {
		meMetrics[s.Metric.String()] = s
	}
	for _, ts := range collection {
		if ts != nil {
			me2 := ts.(*MatrixEnvelope)
			for _, s := range me2.Data.Result {
				name := s.Metric.String()
				if _, ok := meMetrics[name]; !ok {
					meMetrics[name] = s
					me.Data.Result = append(me.Data.Result, s)
					continue
				}
				meMetrics[name].Values = append(meMetrics[name].Values, s.Values...)
			}
			me.ExtentList = append(me.ExtentList, me2.ExtentList...)
		}
	}
	me.ExtentList = me.ExtentList.Compress(me.StepDuration)
	me.isSorted = false
	me.isCounted = false
	if sort {
		me.Sort()
	}
}

// Copy returns a perfect copy of the base Timeseries
func (me *MatrixEnvelope) Copy() timeseries.Timeseries {
	resMe := &MatrixEnvelope{
		isCounted:  me.isCounted,
		isSorted:   me.isSorted,
		tslist:     make(times.Times, len(me.tslist)),
		timestamps: make(map[time.Time]bool),
		Status:     me.Status,
		Data: MatrixData{
			ResultType: me.Data.ResultType,
			Result:     make(model.Matrix, 0, len(me.Data.Result)),
		},
		StepDuration: me.StepDuration,
		ExtentList:   make(timeseries.ExtentList, len(me.ExtentList)),
	}
	copy(resMe.ExtentList, me.ExtentList)
	copy(resMe.tslist, me.tslist)

	for k, v := range me.timestamps {
		resMe.timestamps[k] = v
	}

	for _, ss := range me.Data.Result {
		newSS := &model.SampleStream{Metric: ss.Metric}
		newSS.Values = ss.Values[:]
		resMe.Data.Result = append(resMe.Data.Result, newSS)
	}
	return resMe
}

// CropToSize reduces the number of elements in the Timeseries to the provided count, by evicting elements
// using a least-recently-used methodology. Any timestamps newer than the provided time are removed before
// sizing, in order to support backfill tolerance. The provided extent will be marked as used during crop.
func (me *MatrixEnvelope) CropToSize(sz int, t time.Time, lur timeseries.Extent) {
	me.isCounted = false
	me.isSorted = false
	x := len(me.ExtentList)
	// The Series has no extents, so no need to do anything
	if x < 1 {
		me.Data.Result = model.Matrix{}
		me.ExtentList = timeseries.ExtentList{}
		return
	}

	// Crop to the Backfill Tolerance Value if needed
	if me.ExtentList[x-1].End.After(t) {
		me.CropToRange(timeseries.Extent{Start: me.ExtentList[0].Start, End: t})
	}

	tc := me.TimestampCount()
	if len(me.Data.Result) == 0 || tc <= sz {
		return
	}

	el := timeseries.ExtentListLRU(me.ExtentList).UpdateLastUsed(lur, me.StepDuration)
	sort.Sort(el)

	rc := tc - sz // # of required timestamps we must delete to meet the rentention policy
	removals := make(map[time.Time]bool)
	done := false
	ok := false

	for _, x := range el {
		for ts := x.Start; !x.End.Before(ts) && !done; ts = ts.Add(me.StepDuration) {
			if _, ok = me.timestamps[ts]; ok {
				removals[ts] = true
				done = len(removals) >= rc
			}
		}
		if done {
			break
		}
	}

	for _, s := range me.Data.Result {
		tmp := s.Values[:0]
		for _, r := range s.Values {
			t = r.Timestamp.Time()
			if _, ok := removals[t]; !ok {
				tmp = append(tmp, r)
			}
		}
		s.Values = tmp
	}

	tl := times.FromMap(removals)
	sort.Sort(tl)
	for _, t := range tl {
		for i, e := range el {
			if e.StartsAt(t) {
				el[i].Start = e.Start.Add(me.StepDuration)
			}
		}
	}

	me.ExtentList = timeseries.ExtentList(el).Compress(me.StepDuration)
	me.Sort()
}

// CropToRange reduces the Timeseries down to timestamps contained within the provided Extents (inclusive).
// CropToRange assumes the base Timeseries is already sorted, and will corrupt an unsorted Timeseries
func (me *MatrixEnvelope) CropToRange(e timeseries.Extent) {
	me.isCounted = false
	x := len(me.ExtentList)
	// The Series has no extents, so no need to do anything
	if x < 1 {
		me.Data.Result = model.Matrix{}
		me.ExtentList = timeseries.ExtentList{}
		return
	}

	// if the extent of the series is entirely outside the extent of the crop range, return empty set and bail
	if me.ExtentList.OutsideOf(e) {
		me.Data.Result = model.Matrix{}
		me.ExtentList = timeseries.ExtentList{}
		return
	}

	// if the series extent is entirely inside the extent of the crop range, simply adjust down its ExtentList
	if me.ExtentList.InsideOf(e) {
		if me.ValueCount() == 0 {
			me.Data.Result = model.Matrix{}
		}
		me.ExtentList = me.ExtentList.Crop(e)
		return
	}

	if len(me.Data.Result) == 0 {
		me.ExtentList = me.ExtentList.Crop(e)
		return
	}

	deletes := make(map[int]bool)

	for i, s := range me.Data.Result {
		start := -1
		end := -1
		for j, val := range s.Values {
			t := val.Timestamp.Time()
			if t.Equal(e.End) {
				// for cases where the first element is the only qualifying element,
				// start must be incremented or an empty response is returned
				if j == 0 || t.Equal(e.Start) || start == -1 {
					start = j
				}
				end = j + 1
				break
			}
			if t.After(e.End) {
				end = j
				break
			}
			if t.Before(e.Start) {
				continue
			}
			if start == -1 && (t.Equal(e.Start) || (e.End.After(t) && t.After(e.Start))) {
				start = j
			}
		}
		if start != -1 && len(s.Values) > 0 {
			if end == -1 {
				end = len(s.Values)
			}
			me.Data.Result[i].Values = s.Values[start:end]
		} else {
			deletes[i] = true
		}
	}
	if len(deletes) > 0 {
		tmp := me.Data.Result[:0]
		for i, r := range me.Data.Result {
			if _, ok := deletes[i]; !ok {
				tmp = append(tmp, r)
			}
		}
		me.Data.Result = tmp
	}
	me.ExtentList = me.ExtentList.Crop(e)
}

// Sort sorts all Values in each Series chronologically by their timestamp
func (me *MatrixEnvelope) Sort() {

	if me.isSorted {
		return
	}

	tsm := map[time.Time]bool{}

	for i, s := range me.Data.Result { // []SampleStream
		m := make(map[time.Time]model.SamplePair)
		for _, v := range s.Values { // []SamplePair
			t := v.Timestamp.Time()
			tsm[t] = true
			m[t] = v
		}
		keys := make(times.Times, 0, len(m))
		for key := range m {
			keys = append(keys, key)
		}
		sort.Sort(keys)
		sm := make([]model.SamplePair, 0, len(keys))
		for _, key := range keys {
			sm = append(sm, m[key])
		}
		me.Data.Result[i].Values = sm
	}

	sort.Sort(me.ExtentList)

	me.timestamps = tsm
	me.tslist = times.FromMap(tsm)
	me.isCounted = true
	me.isSorted = true
}

func (me *MatrixEnvelope) updateTimestamps() {
	if me.isCounted {
		return
	}
	m := make(map[time.Time]bool)
	for _, s := range me.Data.Result { // []SampleStream
		for _, v := range s.Values { // []SamplePair
			t := v.Timestamp.Time()
			m[t] = true
		}
	}
	me.timestamps = m
	me.tslist = times.FromMap(m)
	me.isCounted = true
}

// SetExtents overwrites a Timeseries's known extents with the provided extent list
func (me *MatrixEnvelope) SetExtents(extents timeseries.ExtentList) {
	me.isCounted = false
	me.ExtentList = extents
}

// Extents returns the Timeseries's ExentList
func (me *MatrixEnvelope) Extents() timeseries.ExtentList {
	return me.ExtentList
}

// TimestampCount returns the number of unique timestamps across the timeseries
func (me *MatrixEnvelope) TimestampCount() int {
	me.updateTimestamps()
	return len(me.timestamps)
}

// SeriesCount returns the number of individual Series in the Timeseries object
func (me *MatrixEnvelope) SeriesCount() int {
	return len(me.Data.Result)
}

// ValueCount returns the count of all values across all Series in the Timeseries object
func (me *MatrixEnvelope) ValueCount() int {
	c := 0
	for i := range me.Data.Result {
		c += len(me.Data.Result[i].Values)
	}
	return c
}

// methods required for sorting Prometheus model.Times

// Len returns the length of an array of Prometheus model.Times
func (t Times) Len() int {
	return len(t)
}

// Less returns true if i comes before j
func (t Times) Less(i, j int) bool {
	return t[i].Before(t[j])
}

// Swap modifies an array by of Prometheus model.Times swapping the values in indexes i and j
func (t Times) Swap(i, j int) {
	t[i], t[j] = t[j], t[i]
}

// RegisterRoutes registers the routes for the Client into the proxy's HTTP multiplexer
func (c *PromTestClient) RegisterRoutes(originName string, o *config.OriginConfig) {
	routing.Router.HandleFunc("/"+mnHealth, c.HealthHandler).Methods("GET").Host(originName)
	routing.Router.HandleFunc(APIPath+mnQueryRange, c.QueryRangeHandler).Methods("GET", "POST").Host(originName)
	routing.Router.HandleFunc(APIPath+mnQuery, c.QueryHandler).Methods("GET", "POST").Host(originName)
	routing.Router.HandleFunc(APIPath+mnSeries, c.SeriesHandler).Methods("GET", "POST").Host(originName)
	routing.Router.PathPrefix(APIPath).HandlerFunc(c.ProxyHandler).Methods("GET", "POST").Host(originName)
	routing.Router.PathPrefix("/").HandlerFunc(c.ProxyHandler).Methods("GET", "POST").Host(originName)

	// Path based routing
	routing.Router.HandleFunc("/"+originName+"/"+mnHealth, c.HealthHandler).Methods("GET")
	routing.Router.HandleFunc("/"+originName+APIPath+mnQueryRange, c.QueryRangeHandler).Methods("GET", "POST")
	routing.Router.HandleFunc("/"+originName+APIPath+mnQuery, c.QueryHandler).Methods("GET", "POST")
	routing.Router.HandleFunc("/"+originName+APIPath+mnSeries, c.SeriesHandler).Methods("GET", "POST")
	routing.Router.PathPrefix("/"+originName+APIPath).HandlerFunc(c.ProxyHandler).Methods("GET", "POST")
	routing.Router.PathPrefix("/"+originName+"/").HandlerFunc(c.ProxyHandler).Methods("GET", "POST")

	// If default origin, set those routes too
	if o.IsDefault {
		routing.Router.HandleFunc("/"+mnHealth, c.HealthHandler).Methods("GET")
		routing.Router.HandleFunc(APIPath+mnQueryRange, c.QueryRangeHandler).Methods("GET", "POST")
		routing.Router.HandleFunc(APIPath+mnQuery, c.QueryHandler).Methods("GET", "POST")
		routing.Router.HandleFunc(APIPath+mnSeries, c.SeriesHandler).Methods("GET", "POST")
		routing.Router.PathPrefix(APIPath).HandlerFunc(c.ProxyHandler).Methods("GET", "POST")
		routing.Router.PathPrefix("/").HandlerFunc(c.ProxyHandler).Methods("GET", "POST")
	}

}

func (c *PromTestClient) HealthHandler(w http.ResponseWriter, r *http.Request) {
	u := c.BaseURL()
	u.Path += APIPath + mnLabels
	ProxyRequest(tm.NewRequest(c.name, otPrometheus, "HealthHandler", http.MethodGet, u, r.Header, c.config.Timeout, r, c.webClient), w)
}

func (c *PromTestClient) QueryRangeHandler(w http.ResponseWriter, r *http.Request) {
	u := c.BuildUpstreamURL(r)
	DeltaProxyCacheRequest(
		tm.NewRequest(c.name, otPrometheus, "QueryRangeHandler", r.Method, u, r.Header, c.config.Timeout, r, c.webClient),
		w, c, c.cache, c.cache.Configuration().TimeseriesTTL)
}

func (c *PromTestClient) QueryHandler(w http.ResponseWriter, r *http.Request) {
	u := c.BuildUpstreamURL(r)
	ObjectProxyCacheRequest(
		tm.NewRequest(c.name, otPrometheus, "QueryHandler", r.Method, u, r.Header, c.config.Timeout, r, c.webClient),
		w, c, c.cache, c.cache.Configuration().ObjectTTL, false, false)
}

func (c *PromTestClient) SeriesHandler(w http.ResponseWriter, r *http.Request) {
	u := c.BuildUpstreamURL(r)
	ObjectProxyCacheRequest(
		tm.NewRequest(c.name, otPrometheus, "SeriesHandler", r.Method, u, r.Header, c.config.Timeout, r, c.webClient),
		w, c, c.cache, c.cache.Configuration().ObjectTTL, false, false)
}

func (c *PromTestClient) ProxyHandler(w http.ResponseWriter, r *http.Request) {
	ProxyRequest(tm.NewRequest(c.name, otPrometheus, "APIProxyHandler", r.Method, c.BuildUpstreamURL(r), r.Header, c.config.Timeout, r, c.webClient), w)
}

func testResultHeaderPartMatch(header http.Header, kvp map[string]string) error {
	if len(kvp) == 0 {
		return nil
	}
	if header == nil || len(header) == 0 {
		return fmt.Errorf("missing response headers%s", "")
	}

	if h, ok := header["X-Trickster-Result"]; ok {
		res := strings.Join(h, "; ")
		for k, v := range kvp {
			if strings.Index(res, fmt.Sprintf("; %s=%s", k, v)) == -1 && strings.Index(res, fmt.Sprintf("%s=%s", k, v)) != 0 {
				return fmt.Errorf("invalid status, expected %s=%s in %s", k, v, h)
			}
		}
	} else {
		return fmt.Errorf("missing X-Trickster-Result header%s", "")
	}

	return nil
}

func testStatusCodeMatch(have, expected int) error {
	if have != expected {
		return fmt.Errorf("expected http status %d got %d", expected, have)
	}
	return nil
}

func testStringMatch(have, expected string) error {
	if have != expected {
		return fmt.Errorf("expected string `%s` got `%s`", expected, have)
	}
	return nil
}