package envsubst

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

type GetEnvsubstStringFunc func(str string) (string, error)
type GetEnvsubstStringArgsFunc func(args map[string]string, str string) (string, error)
type GetEnvsubstStringArgsFromEnvFunc func(args []string, str string) (string, error)

var GetEnvsubst GetEnvsubstStringFunc = func(str string) (string, error) {
	osCmd := exec.Command("envsubst")
	osCmd.Stdin = strings.NewReader(str)
	osCmd.Stderr = os.Stderr

	outputBytes, err := osCmd.Output()
	return string(outputBytes), err
}

var GetEnvsubstArgs GetEnvsubstStringArgsFunc = func(args map[string]string, str string) (string, error) {
	argsKeys := []string{}
	for key, _ := range args {
		argsKeys = append(argsKeys, fmt.Sprintf("$%s", key))
	}

	osCmd := exec.Command("envsubst", strings.Join(argsKeys, ","))
	osCmd.Env = os.Environ()

	for key, value := range args {
		osCmd.Env = append(osCmd.Env, fmt.Sprintf("%s=%s", key, value))
	}

	osCmd.Stdin = strings.NewReader(str)
	osCmd.Stderr = os.Stderr

	outputBytes, err := osCmd.Output()
	return string(outputBytes), err
}

var GetEnvsubstArgsFromEnv GetEnvsubstStringArgsFromEnvFunc = func(args []string, str string) (string, error) {
	if len(args) == 0 {
		return str, nil
	}

	// If any argument isn't given, return an error
	argsKeys := []string{}
	for _, key := range args {
		_, exists := os.LookupEnv(key)
		if !exists {
			return "", errors.New(fmt.Sprintf("Failed to find %s in environment.", key))
		}
		argsKeys = append(argsKeys, fmt.Sprintf("$%s", key))
	}

	osCmd := exec.Command("envsubst", strings.Join(argsKeys, ","))
	osCmd.Stdin = strings.NewReader(str)
	osCmd.Stderr = os.Stderr

	outputBytes, err := osCmd.Output()
	return string(outputBytes), err
}
