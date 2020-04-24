package common

import (
	"os"
	"path"
	"runtime"
	"strings"
)

// GetCurrentFileLine 현재 파일의 이름을 반환한다.
func GetCurrentFileLine() (string, int) {
	_, filename, line, ok := runtime.Caller(1)
	if !ok {
		panic("No caller information")
	}

	return filename, line
}

// GetCurrentFileName 현재 파일의 이름을 반환한다.
func GetCurrentFileName() string {
	_, filename, _, ok := runtime.Caller(1)
	if !ok {
		panic("No caller information")
	}

	return filename
}

// GetCurrentFilePath 현재 파일의 이름을 반환한다.
func GetCurrentFilePath() string {
	_, filename, _, ok := runtime.Caller(1)
	if !ok {
		panic("No caller information")
	}

	p := path.Dir(filename)
	if p == "" {
		return "."
	}
	return p
}

// GetModuleRootPath 현재 프로젝트 Module의 파일의 이름을 반환한다.
func GetModuleRootPath(childPath string) string {
	_, filename, _, ok := runtime.Caller(1)
	if !ok {
		panic("No caller information")
	}

	p := path.Dir(filename)
	if p == "" {
		return "."
	}
	return strings.Replace(p, childPath, "", 1)
}

// IsExistFile File이 존재하는지 여부블 반환한다.
func IsExistFile(fname string) bool {
	_, err := os.Stat(fname)
	if os.IsNotExist(err) {
		return false
	}
	return err == nil
}
