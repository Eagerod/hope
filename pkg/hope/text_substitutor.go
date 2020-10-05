package hope

import (
	"io/ioutil"
)

import (
	"github.com/Eagerod/hope/pkg/envsubst"
)

type TextSubstitutor struct {
	Bytes *[]byte
}

func NewTextSubstitutorFromBytes(bytes []byte) *TextSubstitutor {
	t := TextSubstitutor{&bytes}
	return &t
}

func NewTextSubstitutorFromString(str string) *TextSubstitutor {
	return NewTextSubstitutorFromBytes([]byte(str))
}

func TextSubstitutorFromFilepath(filepath string) (*TextSubstitutor, error) {
	fileContents, err := ioutil.ReadFile(filepath)
	if err != nil {
		return nil, err
	}

	return NewTextSubstitutorFromBytes(fileContents), nil
}

func (t *TextSubstitutor) SubstituteTextFromEnv(envVarsNames []string) error {
	newBytes, err := envsubst.GetEnvsubstBytes(envVarsNames, *t.Bytes)
	if err != nil {
		return err
	}

	t.Bytes = &newBytes
	return nil
}

func (t *TextSubstitutor) SubstituteTextFromMap(variables map[string]string) error {
	newBytes, err := envsubst.GetEnvsubstBytesArgs(variables, *t.Bytes)
	if err != nil {
		return err
	}

	t.Bytes = &newBytes
	return nil
}
