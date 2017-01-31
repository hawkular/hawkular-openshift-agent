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

package tags

import (
	"fmt"
	"os"
	"testing"

	"github.com/hawkular/hawkular-openshift-agent/util/expand"
)

func TestAppend(t *testing.T) {
	tags := Tags{
		"tag0": "tagvalue0",
	}

	more := map[string]string{
		"another": "anothervalue",
		"lastone": "lastvalue",
	}

	tags.AppendTags(more)

	if len(tags) != 3 {
		t.Fatalf("Failed to append tags: %v", tags)
	}

	assertTagValue(t, tags, "tag0", "tagvalue0")
	assertTagValue(t, tags, "another", "anothervalue")
	assertTagValue(t, tags, "lastone", "lastvalue")
}

func TestExpandWithDefault(t *testing.T) {
	envvar1 := "first envvar"
	defer os.Unsetenv("TEST_FIRST_ENVVAR")
	os.Setenv("TEST_FIRST_ENVVAR", envvar1)

	tags := Tags{
		"tag1": "${TEST_FIRST_ENVVAR=This default is not needed}",
		"tag2": "${THIS_DOES_NOT_EXIST=default value}",
		"tag3": "${THIS_DOES_NOT_EXIST}",
	}

	tags = tags.ExpandTokens(expand.MappingFuncConfig{UseOSEnv: true})

	assertTagValue(t, tags, "tag1", envvar1)
	assertTagValue(t, tags, "tag2", "default value")
	assertTagValue(t, tags, "tag3", "")
}

func TestExpandEnvVars(t *testing.T) {
	envvar1 := "first envvar"
	defer os.Unsetenv("TEST_FIRST_ENVVAR")
	os.Setenv("TEST_FIRST_ENVVAR", envvar1)

	envvar2 := "second envvar"
	defer os.Unsetenv("TEST_SECOND_ENVVAR")
	os.Setenv("TEST_SECOND_ENVVAR", envvar2)

	tags := Tags{
		"tag0": "tagvalue 0 with no tokens!",
		"tag1": "$TEST_FIRST_ENVVAR",
		"tag2": "prefix$TEST_FIRST_ENVVAR",
		"tag3": "${TEST_FIRST_ENVVAR}postfix",
		"tag4": "prefix${TEST_FIRST_ENVVAR}postfix",
		"tag5": "${TEST_FIRST_ENVVAR}${TEST_SECOND_ENVVAR}",
		"tag6": "A${TEST_FIRST_ENVVAR}B${TEST_SECOND_ENVVAR}C",
		"tag7": "$THIS_DOES_NOT_EXIST",
		"tag8": "$$literal",
	}

	tags = tags.ExpandTokens(expand.MappingFuncConfig{UseOSEnv: true})

	assertTagValue(t, tags, "tag0", "tagvalue 0 with no tokens!")
	assertTagValue(t, tags, "tag1", envvar1)
	assertTagValue(t, tags, "tag2", "prefix"+envvar1)
	assertTagValue(t, tags, "tag3", envvar1+"postfix")
	assertTagValue(t, tags, "tag4", "prefix"+envvar1+"postfix")
	assertTagValue(t, tags, "tag5", envvar1+envvar2)
	assertTagValue(t, tags, "tag6", fmt.Sprintf("A%vB%vC", envvar1, envvar2))
	assertTagValue(t, tags, "tag7", "")
	assertTagValue(t, tags, "tag8", "$literal")
}

func TestAdditionalValues(t *testing.T) {

	tags := Tags{"tag1": "$not_a_env_var"}
	tags = tags.ExpandTokens(expand.MappingFuncConfig{Env: map[string]string{"not_a_env_var": "some value"}})
	assertTagValue(t, tags, "tag1", "some value")

	tags = Tags{"tag1": "the sum of $one plus $two is ${three}"}
	tags = tags.ExpandTokens(expand.MappingFuncConfig{
		Env: map[string]string{
			"one":    "1",
			"two":    "2",
			"three":  "3",
			"unused": "?",
		},
	})
	assertTagValue(t, tags, "tag1", "the sum of 1 plus 2 is 3")
}

func TestSpecialCharsInNames(t *testing.T) {

	tags := Tags{"tag1": "pod name is ${POD:Name} p|a = ${p|a}"}
	tags = tags.ExpandTokens(expand.MappingFuncConfig{
		Env: map[string]string{
			"POD:Name": "foo",
			"p|a":      "bar",
		},
	})
	assertTagValue(t, tags, "tag1", "pod name is foo p|a = bar")
}

func TestOverrideEnvVar(t *testing.T) {
	envvar1 := "first envvar"
	defer os.Unsetenv("TEST_FIRST_ENVVAR")
	os.Setenv("TEST_FIRST_ENVVAR", envvar1)

	tags := Tags{"tag1": "$TEST_FIRST_ENVVAR"}
	tags = tags.ExpandTokens(expand.MappingFuncConfig{UseOSEnv: true})
	assertTagValue(t, tags, "tag1", envvar1)

	tags = Tags{"tag1": "$TEST_FIRST_ENVVAR"}
	tags = tags.ExpandTokens(expand.MappingFuncConfig{})
	assertTagValue(t, tags, "tag1", "")

	tags = Tags{"tag1": "$TEST_FIRST_ENVVAR"}
	tags = tags.ExpandTokens(expand.MappingFuncConfig{
		UseOSEnv: true,
		Env:      map[string]string{"TEST_FIRST_ENVVAR": "override value"},
	})
	assertTagValue(t, tags, "tag1", "override value")

	tags = Tags{"tag1": "$TEST_FIRST_ENVVAR"}
	tags = tags.ExpandTokens(expand.MappingFuncConfig{
		Env: map[string]string{"TEST_FIRST_ENVVAR": "override value"},
	})
	assertTagValue(t, tags, "tag1", "override value")
}

func assertTagValue(t *testing.T, tags Tags, key string, expected string) {
	if tags[key] != expected {
		t.Fatalf("Tag [%v] should have been [%v] but was [%v]", key, expected, tags[key])
	}
}
