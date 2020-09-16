package envsubst

import (
	"os"
	"os/exec"
	"strings"
)

type GetEnvsubstStringFunc func(str string) (string, error)

var GetEnvsubst GetEnvsubstStringFunc = func(str string) (string, error) {
	osCmd := exec.Command("envsubst")
	osCmd.Stdin = strings.NewReader(str)
	osCmd.Stderr = os.Stderr

	outputBytes, err := osCmd.Output()
	return string(outputBytes), err
}
