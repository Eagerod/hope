package envsubst

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

type GetEnvsubstStringFunc func(str string) (string, error)
type GetEnvsubstStringArgsFunc func(args map[string]string, str string) (string, error)
type GetEnvsubstBytesArgsFunc func(args map[string]string, bytes []byte) ([]byte, error)
type GetEnvsubstBytesArgsFromEnvFunc func(args []string, bytes []byte) ([]byte, error)
type GetEnvsubstStringArgsFromEnvFunc func(args []string, str string) (string, error)

var GetEnvsubst GetEnvsubstStringFunc = func(str string) (string, error) {
	osCmd := exec.Command("envsubst")
	osCmd.Stdin = strings.NewReader(str)
	osCmd.Stderr = os.Stderr

	outputBytes, err := osCmd.Output()
	return string(outputBytes), err
}

var GetEnvsubstBytes GetEnvsubstBytesArgsFromEnvFunc = func(args []string, contents []byte) ([]byte, error) {
	if len(args) == 0 {
		return contents, nil
	}

	// If any argument isn't given, return an error
	argsKeys := []string{}
	for _, key := range args {
		_, exists := os.LookupEnv(key)
		if !exists {
			return []byte{}, errors.New(fmt.Sprintf("Failed to find %s in environment.", key))
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


var GetEnvsubstArgs GetEnvsubstStringArgsFunc = func(args map[string]string, str string) (string, error) {
	outputBytes, err := GetEnvsubstBytesArgs(args, []byte(str))
	if err != nil {
		return "", err
	}

	return string(outputBytes), nil
}

var GetEnvsubstArgsFromEnv GetEnvsubstStringArgsFromEnvFunc = func(args []string, str string) (string, error) {
	outputBytes, err := GetEnvsubstBytes(args, []byte(str))
	if err != nil {
		return "", err
	}

	return string(outputBytes), nil
}
