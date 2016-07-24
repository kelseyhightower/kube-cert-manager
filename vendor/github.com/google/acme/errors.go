// Copyright 2015 Google Inc. All Rights Reserved.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//     http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package acme

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"
)

// Predefined Error.Type values by the ACME spec.
const (
	ErrBadCSR       = "urn:acme:error:badCSR"
	ErrBadNonce     = "urn:acme:error:badNonce"
	ErrConnection   = "urn:acme:error:connection"
	ErrDNSSec       = "urn:acme:error:dnssec"
	ErrMalformed    = "urn:acme:error:malformed"
	ErrInternal     = "urn:acme:error:serverInternal"
	ErrTLS          = "urn:acme:error:tls"
	ErrUnauthorized = "urn:acme:error:unauthorized"
	ErrUnknownHost  = "urn:acme:error:unknownHost"
	ErrRateLimited  = "urn:acme:error:rateLimited"
)

// Error is an ACME error.
type Error struct {
	Status int
	Type   string
	Detail string
	// Response is the original server response used to construct the Error,
	// with Response.Body closed.
	Response *http.Response `json:"-"`
}

func (e *Error) Error() string {
	return fmt.Sprintf("%d %s: %s", e.Status, e.Type, e.Detail)
}

// responseError creates an error of Error type from resp.
func responseError(resp *http.Response) error {
	// don't care if ReadAll returns an error:
	// json.Unmarshal will fail in that case anyway
	b, _ := ioutil.ReadAll(resp.Body)
	e := &Error{Status: resp.StatusCode, Response: resp}
	if err := json.Unmarshal(b, e); err != nil {
		// this is not a regular error response:
		// populate detail with anything we received,
		// e.Status will already contain HTTP response code value
		e.Detail = string(b)
		if e.Detail == "" {
			e.Detail = resp.Status
		}
	}
	return e
}

// RetryError is a "temporary" error indicating that the request
// can be retried after the specified duration.
type RetryError time.Duration

func (re RetryError) Error() string {
	return fmt.Sprintf("retry after %s", re)
}
