/*
   Copyright 2016 Red Hat, Inc. and/or its affiliates
   and other contributors.

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

       http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

package security

import (
	"testing"
)

func TestValidateCredentials(t *testing.T) {
	var creds *Credentials

	creds = &Credentials{}
	if err := creds.ValidateCredentials(); err != nil {
		t.Errorf("Empty credentials should be valid: %v", err)
	}

	if headerName, headerValue, err := creds.GetHttpAuthHeader(); err != nil {
		t.Errorf("Should not have received error: %v", err)
	} else {
		if headerName != "" || headerValue != "" {
			t.Errorf("Bad auth header: %v=%v", headerName, headerValue)
		}
	}

	creds = &Credentials{
		Username: "u",
		Password: "p",
	}
	if err := creds.ValidateCredentials(); err != nil {
		t.Errorf("Username/Password credentials should be valid: %v", err)
	}

	if headerName, headerValue, err := creds.GetHttpAuthHeader(); err != nil {
		t.Errorf("Should not have received error: %v", err)
	} else {
		if headerName != "Authorization" || headerValue != "Basic dTpw" {
			t.Errorf("Bad auth header for %v: %v=%v", creds, headerName, headerValue)
		}
	}

	creds = &Credentials{
		Token: "t",
	}
	if err := creds.ValidateCredentials(); err != nil {
		t.Errorf("Token credentials should be valid: %v", err)
	}

	if headerName, headerValue, err := creds.GetHttpAuthHeader(); err != nil {
		t.Errorf("Should not have received error: %v", err)
	} else {
		if headerName != "Authorization" || headerValue != "Bearer t" {
			t.Errorf("Bad auth header for %v: %v=%v", creds, headerName, headerValue)
		}
	}

	creds = &Credentials{
		Username: "u",
		Password: "p",
		Token:    "t",
	}
	if err := creds.ValidateCredentials(); err == nil {
		t.Error("Setting both Username/Password and Token should be invalid")
	}

	creds = &Credentials{
		Username: "u",
	}
	if err := creds.ValidateCredentials(); err == nil {
		t.Error("Setting Username without Password should be invalid")
	}

	creds = &Credentials{
		Password: "p",
	}
	if err := creds.ValidateCredentials(); err == nil {
		t.Error("Setting Password without Username should be invalid")
	}
}
