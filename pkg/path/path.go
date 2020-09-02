package path

import (
	"strings"
)

import (
	homedir "github.com/mitchellh/go-homedir"
)

func ExpandHome(localPath string) (string, error) {
	if strings.HasPrefix(localPath, "~") {
		home, err := homedir.Dir()
		if err != nil {
			return "", err
		}

		localPath = strings.Replace(localPath, "~", home, 1)
	}

	return localPath, nil
}
