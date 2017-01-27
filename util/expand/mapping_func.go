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
	"strings"
)

// MappingFuncConfig details how the mapping function is to be configured
// when created by the MappingFunc function.
//
// Env contains the variables to be used for looking up and expanding the ${x} tokens.
// If UseOSEnv is true, the OS environment variables are also used during the lookup.
// If DoNotExpandIfNotFound is false, and a match is not found in the environment,
// the default value is an empty string. If this is true, the value will remain ${x}.
// Note that if the value seen was $x, it will be replaced with ${x}.
// This is ignored if a default was explicitly configured as in ${x=default}
// in which case the specified default is used regardless.
type MappingFuncConfig struct {
	Env                   map[string]string
	UseOSEnv              bool
	DoNotExpandIfNotFound bool
}

// MappingFunc returns a mapping function for use with os.Expand.
// It will expand token expressions such as $name or ${name}, replacing
// them with their corresponding values found in either
// the operating system environment variable table and/or the given
// additional environment map as declared in the config object.
//
// A default value can be optionally specified in the following as ${name=default}.
// If a default value is not specified, an empty string is used as the default
// unless the config indicates no expansion should be performed.
//
// If a name is found in both the OS environment and config environment, the
// config environment value will be used to replace the $name token.
// If you want a literal $ in the string, use $$.
func MappingFunc(config MappingFuncConfig) func(s string) string {
	theMappingFunc := func(s string) string {
		if s == "$" {
			return "$" // a $$ means the user wants a literal "$" character
		}

		defaultVal := ""
		if config.DoNotExpandIfNotFound {
			defaultVal = "${" + s + "}"
		}

		// Strip off any default value that was provided.
		nameAndDefault := strings.SplitN(s, "=", 2)
		if len(nameAndDefault) == 2 {
			s = nameAndDefault[0]
			defaultVal = nameAndDefault[1]
		}

		// Look up the value, first in the additional env map, then in the OS env map
		if val, ok := config.Env[s]; ok {
			return val
		}

		if config.UseOSEnv {
			if val, ok := os.LookupEnv(s); ok {
				return val
			}
		}

		return defaultVal
	}

	return theMappingFunc
}
