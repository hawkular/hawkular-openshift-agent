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
	"os"

	"github.com/hawkular/hawkular-openshift-agent/util/expand"
)

// Identifies a list of name=value tags
// USED FOR YAML
type Tags map[string]string

func (t *Tags) AppendTags(moreTags map[string]string) {
	if moreTags != nil && len(moreTags) > 0 {
		for k, v := range moreTags {
			(*t)[k] = v
		}
	}
}

// ExpandTokens will replace all tag values such that $name or ${name}
// expressions are replaced with their corresponding values found in either
// the operating system environment variable table and/or the given
// additional environment map. The expanded tags map is returned.
// See docs on expand.MappingFunc for more details.
func (t *Tags) ExpandTokens(config expand.MappingFuncConfig) map[string]string {
	if t == nil {
		return map[string]string{}
	}

	mappingFunc := expand.MappingFunc(config)

	ret := make(map[string]string, len(*t))

	for k, v := range *t {
		ret[k] = os.Expand(v, mappingFunc)
	}

	return ret
}
