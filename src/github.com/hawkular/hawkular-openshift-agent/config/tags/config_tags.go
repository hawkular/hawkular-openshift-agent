package tags

import (
	"os"
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
// additional environment map.
// If useOsEnv is false, the OS environment variables are not used.
// If additionalEnv is nil, it is ignored.
// If a name is found in both the OS environment and additionalEnv, the
// additionalEnv value will be used to replace the $name token.
// If a name is not found, an empty string is used to replace the $name token.
// If you want a literal $ in the string, use $$.
func (t *Tags) ExpandTokens(useOsEnv bool, additionalEnv *map[string]string) {
	if t == nil {
		return
	}

	mappingFunc := func(s string) string {
		if s == "$" {
			return "$" // a $$ means the user wants a literal "$" character
		}

		if additionalEnv != nil {
			if val, ok := (*additionalEnv)[s]; ok {
				return val
			}
		}

		if useOsEnv {
			if val, ok := os.LookupEnv(s); ok {
				return val
			}
		}

		return ""
	}

	for k, v := range *t {
		(*t)[k] = os.Expand(v, mappingFunc)
	}
}
