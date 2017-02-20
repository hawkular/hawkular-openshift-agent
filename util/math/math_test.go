/*
   Copyright 2017 Red Hat, Inc. and/or its affiliates
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

package math

import (
	"math"
	"testing"
)

func TestAvg(t *testing.T) {
	assertEquals(t, 2.5, Avg([]float64{1.0, 2.5, 4.0}), "Avg is broken")
	assertEquals(t, 0.0, Avg([]float64{}), "Avg is broken")
}

func TestStddev(t *testing.T) {
	arr := []float64{1.0, 1.0, 1.0}
	assertEquals(t, 0.0, Stddev(arr, Avg(arr)), "Stddev is broken")
	arr = []float64{6.0, 2.0, 3.0, 1.0}
	assertEquals(t, 1.87, Stddev(arr, Avg(arr)), "Stddev is broken")
	assertEquals(t, 0.0, Stddev([]float64{}, 0.0), "Stddev is broken")
}

func TestSum(t *testing.T) {
	assertEquals(t, 10.5, Sum([]float64{1.0, 2.5, 3.5, 3.5}), "Sum is broken")
	assertEquals(t, 0.0, Sum([]float64{}), "Sum is broken")
}

func TestMin(t *testing.T) {
	assertEquals(t, -1.25, Min([]float64{2.0, -1.25, 4.5, 5.5}), "Min is broken")
	assertEquals(t, 2.5, Min([]float64{2.5, 4.5, 5.5}), "Min is broken")
	assertEquals(t, 1.5, Min([]float64{1.5, math.NaN(), 5.5}), "Min is broken")
	assertEquals(t, 0.0, Min([]float64{}), "Min is broken")
	if !math.IsNaN(Min([]float64{math.NaN(), math.NaN()})) {
		t.Errorf("Min is broken")
	}
	if !math.IsInf(Min([]float64{math.Inf(1)}), 1) {
		t.Errorf("Min is broken")
	}
	if !math.IsInf(Min([]float64{math.Inf(-1)}), -1) {
		t.Errorf("Min is broken")
	}
	if !math.IsInf(Min([]float64{math.Inf(-1), 1.0}), -1) {
		t.Errorf("Min is broken")
	}
	assertEquals(t, 1.5, Min([]float64{math.Inf(1), 1.5}), "Min is broken")
}

func TestMax(t *testing.T) {
	assertEquals(t, -1.25, Max([]float64{-2.0, -1.25, -4.5, -5.5}), "Max is broken")
	assertEquals(t, 5.5, Max([]float64{2.0, 4.5, 5.5}), "Max is broken")
	assertEquals(t, 5.5, Max([]float64{1.0, math.NaN(), 5.5}), "Max is broken")
	assertEquals(t, 0.0, Max([]float64{}), "Max is broken")
	if !math.IsNaN(Max([]float64{math.NaN(), math.NaN()})) {
		t.Errorf("Max is broken")
	}
	if !math.IsInf(Max([]float64{math.Inf(1)}), 1) {
		t.Errorf("Max is broken")
	}
	if !math.IsInf(Max([]float64{math.Inf(-1)}), -1) {
		t.Errorf("Max is broken")
	}
	if !math.IsInf(Max([]float64{math.Inf(1), 1.0}), 1) {
		t.Errorf("Max is broken")
	}
	assertEquals(t, 1.5, Max([]float64{math.Inf(-1), 1.5}), "Max is broken")
}

func assertEquals(t *testing.T, expected float64, actual float64, msg string) {
	if diff := math.Abs(expected - actual); diff > 0.001 {
		t.Errorf("%v: expected=[%v], actual=[%v], diff=[%v]", msg, expected, actual, diff)
	}
}
