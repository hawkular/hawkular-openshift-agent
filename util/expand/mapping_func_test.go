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

package expand

import (
	"os"
	"testing"
)

func TestMappingFunc(t *testing.T) {
	theFunc := MappingFunc(true, nil)

	envvar1 := "first envvar"
	defer os.Unsetenv("TEST_FIRST_ENVVAR")
	os.Setenv("TEST_FIRST_ENVVAR", envvar1)

	var s string

	s = os.Expand("${TEST_FIRST_ENVVAR=This default is not needed}", theFunc)
	if s != envvar1 {
		t.Fatalf("Bad expansion: %v", s)
	}

	s = os.Expand("${THIS_DOES_NOT_EXIST=default value}", theFunc)
	if s != "default value" {
		t.Fatalf("Bad expansion: %v", s)
	}

	s = os.Expand("${THIS_DOES_NOT_EXIST}", theFunc)
	if s != "" {
		t.Fatalf("Bad expansion: %v", s)
	}
}

func TestMappingFuncAdditionalValues(t *testing.T) {
	var s string

	theFunc := MappingFunc(true, map[string]string{"not_a_env_var": "some value"})
	s = os.Expand("$not_a_env_var", theFunc)
	if s != "some value" {
		t.Fatalf("Bad expansion: %v", s)
	}

	theFunc = MappingFunc(true, map[string]string{
		"one":    "1",
		"two":    "2",
		"three":  "3",
		"unused": "?",
	})
	s = os.Expand("the sum of $one plus $two is ${three}", theFunc)
	if s != "the sum of 1 plus 2 is 3" {
		t.Fatalf("Bad expansion: %v", s)
	}
}
