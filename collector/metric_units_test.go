/*
   Copyright 2016-2017 Red Hat, Inc. and/or its affiliates
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

package collector

import (
	"testing"
)

func TestGetMetricUnits(t *testing.T) {
	// make sure valid metrics are retrieved successfully
	u, e := GetMetricUnits("ms")
	if e != nil {
		t.Errorf("Should not have failed. Error=%v", e)
	}
	if u.Symbol != "ms" {
		t.Errorf("Should have matched 'ms'. u=%v", u)
	}
	if u.Custom != false {
		t.Errorf("Should not have been custom. u=%v", u)
	}

	u, e = GetMetricUnits("custom:foobars")
	if e != nil {
		t.Errorf("Should not have failed. Error=%v", e)
	}
	if u.Symbol != "foobars" {
		t.Errorf("Should have matched 'foobars'. u=%v", u)
	}
	if u.Custom != true {
		t.Errorf("Should have been custom. u=%v", u)
	}

	u, e = GetMetricUnits("")
	if e != nil {
		t.Errorf("Should not have failed. Empty units means 'none'. Error=%v", e)
	}
	if u.Symbol != "" {
		t.Errorf("Should have matched the 'none' units (an empty string). u=%v", u)
	}
	if u.Custom != false {
		t.Errorf("Should not have been custom. u=%v", u)
	}

	// make sure errors are generated properly
	u, e = GetMetricUnits("millis")
	if e == nil {
		t.Errorf("Should have an error - not a standard metric. u=%v", u)
	}
	if u.Symbol != "Unknown" {
		t.Errorf("Should have matched the 'Unknown' unit. u=%v", u)
	}
	if u.Custom != false {
		t.Errorf("Should not have been custom. u=%v", u)
	}

	u, e = GetMetricUnits("foobars")
	if e == nil {
		t.Errorf("Should have an error - not a standard metric. u=%v", u)
	}
	if u.Symbol != "Unknown" {
		t.Errorf("Should have matched the 'Unknown' unit. u=%v", u)
	}
	if u.Custom != false {
		t.Errorf("Should not have been custom. u=%v", u)
	}
}
