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
		t.Errorf("Should have failed - not a standard metric. u=%v", u)
	}
	u, e = GetMetricUnits("foobars")
	if e == nil {
		t.Errorf("Should have failed - not a standard metric. u=%v", u)
	}
}
