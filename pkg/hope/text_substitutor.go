package hope

import (
	"os"
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
	fileContents, err := os.ReadFile(filepath)
	if err != nil {
		return nil, err
	}

	return NewTextSubstitutorFromBytes(fileContents), nil
}

func (t *TextSubstitutor) SubstituteTextFromEnv(envVarsNames []string) error {
	// Don't even hit the downstream tool if we aren't trying to populate
	//   anything.
	if len(envVarsNames) == 0 {
		return nil
	}

	newBytes, err := envsubst.GetEnvsubstBytes(envVarsNames, *t.Bytes)
	if err != nil {
		return err
	}

	t.Bytes = &newBytes
	return nil
}

func (t *TextSubstitutor) SubstituteTextFromMap(variables map[string]string) error {
	// Don't even hit the downstream tool if we aren't trying to populate
	//   anything.
	if len(variables) == 0 {
		return nil
	}

	newBytes, err := envsubst.GetEnvsubstBytesArgs(variables, *t.Bytes)
	if err != nil {
		return err
	}

	t.Bytes = &newBytes
	return nil
}
