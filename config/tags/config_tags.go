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

package tags

import (
	"os"
	"strings"
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
//
// A default value can be optionally specified in the following manner:
//    ${name=default}
// If a default value is not specified, an empty string is used as the default.
//
// If useOsEnv is false, the OS environment variables are not used.
// If additionalEnv is nil, it is ignored.
// If a name is found in both the OS environment and additionalEnv, the
// additionalEnv value will be used to replace the $name token.
// If a name is not found, the default value is used to replace the $name token.
// If you want a literal $ in the string, use $$.
func (t *Tags) ExpandTokens(useOsEnv bool, additionalEnv map[string]string) map[string]string {
	if t == nil {
		return map[string]string{}
	}

	mappingFunc := func(s string) string {
		if s == "$" {
			return "$" // a $$ means the user wants a literal "$" character
		}

		defaultVal := ""

		// Strip off any default value that was provided.
		nameAndDefault := strings.SplitN(s, "=", 2)
		if len(nameAndDefault) == 2 {
			s = nameAndDefault[0]
			defaultVal = nameAndDefault[1]
		}

		// Look up the value, first in the additional env map, then in the OS env map
		if val, ok := additionalEnv[s]; ok {
			return val
		}

		if useOsEnv {
			if val, ok := os.LookupEnv(s); ok {
				return val
			}
		}

		return defaultVal
	}

	ret := make(map[string]string, len(*t))

	for k, v := range *t {
		ret[k] = os.Expand(v, mappingFunc)
	}

	return ret
}
