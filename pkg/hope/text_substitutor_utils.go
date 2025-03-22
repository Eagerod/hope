// Package utils -- Utility functions to be used throughout cmd.
// Kind of decided that this was a pretty decent way of doing shared logic for
// cmd.
// Basically the same as for pkg, so nothing too special there, but since cmd
// has a bit of nesting, it brings up some more questions.
// It seems like golang itself does a pattern like this for `base` cmd units,
// so this is probably not too shabby.
package hope

import (
	"os"
	"path/filepath"
	"strings"
)

import (
	"github.com/Eagerod/hope/pkg/fileutil"
)

func ReplaceParametersInString(str string, parameters []string) (string, error) {
	t := NewTextSubstitutorFromString(str)
	return ReplaceParametersWithSubstitutor(t, parameters)
}

func ReplaceParametersInFile(path string, parameters []string) (string, error) {
	t, err := TextSubstitutorFromFilepath(path)
	if err != nil {
		return "", err
	}

	return ReplaceParametersWithSubstitutor(t, parameters)
}

// ReplaceParametersInFileCopy - Copy the provided file to a temp file, and
// replace parameters in the files.
// Returns the temp path to the copied file, and the caller must clean up
// that file itself, unless an error occurs.
func ReplaceParametersInFileCopy(path string, parameters []string) (string, error) {
	str, err := ReplaceParametersInFile(path, parameters)
	if err != nil {
		return "", err
	}

	tf, err := os.CreateTemp("", "")
	if err != nil {
		return "", err
	}

	if _, err := tf.WriteString(str); err != nil {
		return "", err
	}

	return tf.Name(), nil
}

func ReplaceParametersWithSubstitutor(t *TextSubstitutor, parameters []string) (string, error) {
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

func replaceParametersInDirectory(dir string, parameters []string) error {
	return filepath.Walk(dir, func(apath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		str, err := ReplaceParametersInFile(apath, parameters)
		if err != nil {
			return err
		}

		return fileutil.WriteFile(str, apath)
	})
}

// ReplaceParametersInDirectoryCopy - Copy the provided directory, and replace
// parameters in the files.
// Returns the temp path to the copied directory, and the caller must clean up
// that directory itself, unless an error occurs.
func ReplaceParametersInDirectoryCopy(dir string, parameters []string) (string, error) {
	tempDir, err := os.MkdirTemp("", "*")
	if err != nil {
		return "", err
	}

	if err := fileutil.CopyDirectory(dir, tempDir); err != nil {
		os.RemoveAll(tempDir)
		return "", err
	}

	if len(parameters) != 0 {
		if err := replaceParametersInDirectory(tempDir, parameters); err != nil {
			os.RemoveAll(tempDir)
			return "", err
		}
	}

	return tempDir, nil
}
