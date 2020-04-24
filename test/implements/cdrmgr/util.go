package cdr

import (
	"fmt"
	"io"
	"os"
)

const (
	BYTE     = 1
	KILOBYTE = 1024 * BYTE
	MEGABYTE = 1024 * KILOBYTE
	GIGABYTE = 1024 * MEGABYTE
	TERABYTE = 1024 * GIGABYTE
)

func get_file_size(path string) (bool, error) {

	loggers.InfoLogger().Comment("File_Size_check : %s", path)
	stat, err := os.Stat(path)
	if err != nil {
		loggers.ErrorLogger().Major("CDR file size get Fail - file Not Exist(%s)\nReason : %s",
			path, err.Error())
		return false, err
	}

	check := CheckByteSize(uint64(stat.Size()))
	return check, err
}

func CheckByteSize(bytes uint64) bool {
	unit := ""
	check := false
	//value := float32(bytes)
	//value := int(bytes)

	switch {

	case bytes >= TERABYTE:
		unit = "T"
		bytes = bytes / TERABYTE
	case bytes >= GIGABYTE:
		unit = "G"
		bytes = bytes / GIGABYTE
	case bytes >= MEGABYTE:
		unit = "M"
		bytes = bytes / MEGABYTE
		check = true
	case bytes >= KILOBYTE:
		unit = "K"
		bytes = bytes / KILOBYTE
	default: /* case bytes >= BYTE: case bytes == 0:*/
		unit = "B"
	}

	stringValue := fmt.Sprintf("%d", bytes)
	strFileSize := fmt.Sprintf("%s%s", stringValue, unit)
	loggers.InfoLogger().Comment("File Size : %s", strFileSize)

	return check

}

func IsEmpty(name string) (bool, error) {
	f, err := os.Open(name)
	if err != nil {
		return false, err
	}
	defer f.Close()

	_, err = f.Readdirnames(1) // Or f.Readdir(1)
	if err == io.EOF {
		return true, nil
	}
	return false, err // Either not empty or error, suits both cases
}
