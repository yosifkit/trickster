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

package urls

import (
	"net/url"
	"testing"
)

func TestClone(t *testing.T) {

	u1, _ := url.Parse("http://user:pass@127.0.0.1:8080/path?param1=param2")
	u2 := Clone(u1)
	if u2.Hostname() != "127.0.0.1" {
		t.Errorf("expected %s got %s", "127.0.0.1", u2.Hostname())
	}

	u1, _ = url.Parse("http://user@127.0.0.1:8080/path?param1=param2")
	u2 = Clone(u1)
	if u2.Hostname() != "127.0.0.1" {
		t.Errorf("expected %s got %s", "127.0.0.1", u2.Hostname())
	}

}
