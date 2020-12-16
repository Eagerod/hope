// Package utils -- Utility functions to be used throughout cmd.
// Kind of decided that this was a pretty decent way of doing shared logic for
//   cmd.
// Basically the same as for pkg, so nothing too special there, but since cmd
//   has a bit of nesting, it brings up some more questions.
// It seems like golang itself does a pattern like this for `base` cmd units,
//   so this is probably not too shabby.
package utils

import (
	"strings"
)

import (
	"github.com/Eagerod/hope/pkg/hope"
)

func ReplaceParametersInString(str string, parameters []string) (string, error) {
	t := hope.NewTextSubstitutorFromString(str)
	return ReplaceParametersWithSubstitutor(t, parameters)
}

func ReplaceParametersInFile(path string, parameters []string) (string, error) {
	t, err := hope.TextSubstitutorFromFilepath(path)
	if err != nil {
		return "", err
	}

	return ReplaceParametersWithSubstitutor(t, parameters)
}

func ReplaceParametersWithSubstitutor(t *hope.TextSubstitutor, parameters []string) (string, error) {
	envParams := []string{}
	directParams := map[string]string{}
	for _, value := range parameters {
		parts := strings.SplitN(value, "=", 2)
		if len(parts) == 1 {
			envParams = append(envParams, value)
		} else {
			directParams[parts[0]] = parts[1]
		}
	}

	if err := t.SubstituteTextFromEnv(envParams); err != nil {
		return "", err
	}

	if err := t.SubstituteTextFromMap(directParams); err != nil {
		return "", err
	}

	return string(*t.Bytes), nil
}
