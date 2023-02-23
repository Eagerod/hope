package fileutil

import (
	"crypto/md5"
	"fmt"
	"io"
	"os"
)

func CopyFileMode(src string, dst string, fileMode os.FileMode) error {
	if _, err := os.Stat(dst); !os.IsNotExist(err) {
		return fmt.Errorf("refusing to overwrite existing file (%s)", dst)
	}

	srcFile, err := os.ReadFile(src)
	if err != nil {
		return err
	}

	err = os.WriteFile(dst, srcFile, fileMode)
	if err != nil {
		return err
	}

	return nil
}

func CopyFile(src string, dst string) error {
	return CopyFileMode(src, dst, 0644)
}

func WriteFileMode(str string, dst string, fileMode os.FileMode) error {
	bytes := []byte(str)
	return os.WriteFile(dst, bytes, fileMode)
}

func WriteFile(str string, dst string) error {
	return WriteFileMode(str, dst, 0644)
}

func MD5Path(path string) ([]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return []byte{}, err
	}
	defer file.Close()

	hash := md5.New()
	if _, err := io.Copy(hash, file); err != nil {
		return []byte{}, err
	}

	return hash.Sum(nil), nil
}

func FilesIdentical(f1 string, f2 string) (bool, error) {
	f1Stat, err := os.Stat(f1)
	if err != nil {
		return false, err
	}

	f2Stat, err := os.Stat(f2)
	if err != nil {
		return false, err
	}

	if f1Stat.Size() != f2Stat.Size() {
		return false, nil
	}

	f1Hash, err := MD5Path(f1)
	if err != nil {
		return false, err
	}

	f2Hash, err := MD5Path(f2)
	if err != nil {
		return false, err
	}

	return string(f1Hash) == string(f2Hash), nil
}
