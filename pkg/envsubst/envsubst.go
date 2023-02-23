package envsubst

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

type GetEnvsubstBytesArgsFunc func(args map[string]string, bytes []byte) ([]byte, error)
type GetEnvsubstBytesArgsFromEnvFunc func(args []string, bytes []byte) ([]byte, error)

var GetEnvsubstBytes GetEnvsubstBytesArgsFromEnvFunc = func(args []string, contents []byte) ([]byte, error) {
	if len(args) == 0 {
		return contents, nil
	}

	// If any argument isn't given, return an error
	argsKeys := []string{}
	for _, key := range args {
		_, exists := os.LookupEnv(key)
		if !exists {
			return []byte{}, fmt.Errorf("failed to find %s in environment", key)
		}
		argsKeys = append(argsKeys, fmt.Sprintf("$%s", key))
	}

	osCmd := exec.Command("envsubst", strings.Join(argsKeys, ","))
	osCmd.Stdin = bytes.NewReader(contents)
	osCmd.Stderr = os.Stderr

	return osCmd.Output()
}

var GetEnvsubstBytesArgs GetEnvsubstBytesArgsFunc = func(args map[string]string, contents []byte) ([]byte, error) {
	argsKeys := []string{}
	for key, _ := range args {
		argsKeys = append(argsKeys, fmt.Sprintf("$%s", key))
	}

	osCmd := exec.Command("envsubst", strings.Join(argsKeys, ","))
	osCmd.Env = os.Environ()

	for key, value := range args {
		osCmd.Env = append(osCmd.Env, fmt.Sprintf("%s=%s", key, value))
	}

	osCmd.Stdin = bytes.NewReader(contents)
	osCmd.Stderr = os.Stderr

	return osCmd.Output()
}
