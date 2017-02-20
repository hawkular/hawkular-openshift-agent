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
)

func Min(numbers []float64) float64 {
	if len(numbers) == 0 {
		return 0.0
	}
	m := math.NaN()
	for _, n := range numbers {
		if math.IsNaN(m) || n < m {
			m = n
		}
	}
	return m
}

func Max(numbers []float64) float64 {
	if len(numbers) == 0 {
		return 0.0
	}
	m := math.NaN()
	for _, n := range numbers {
		if math.IsNaN(m) || n > m {
			m = n
		}
	}
	return m
}

func Sum(numbers []float64) float64 {
	total := 0.0
	for _, n := range numbers {
		total += n
	}
	return total
}

func Avg(numbers []float64) float64 {
	if len(numbers) == 0 {
		return 0.0
	}
	return Sum(numbers) / float64(len(numbers))
}

// Yes, this func could calc the mean itself, but if the caller already did that, no sense in doing it again.
// Let the caller pass in Avg(numbers) if mean is not yet calculated.
func Stddev(numbers []float64, mean float64) float64 {
	if len(numbers) == 0 {
		return 0.0
	}
	total := 0.0
	for _, n := range numbers {
		total += math.Pow(n-mean, 2)
	}
	v := total / float64(len(numbers))
	return math.Sqrt(v)
}
