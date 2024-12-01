package main

import (
	"fmt"
	"log"
	"path/filepath"
	"strings"
)

type IncludesNormalize struct {
	relative_to_       string
	split_relative_to_ []string
}

// / Normalize path relative to |relative_to|.
func NewIncludesNormalize(relative_to string) *IncludesNormalize {
	ret := IncludesNormalize{}
	err := ""
	ret.relative_to_ = AbsPath(relative_to, &err)
	if err != "" {
		log.Fatalf("Initializing IncludesNormalize(): %s", err)
	}
	ret.split_relative_to_ = strings.Split(ret.relative_to_, "/")
	return &ret
}

// Internal utilities made available for testing, maybe useful otherwise.
func AbsPath(s string, err1 *string) string {
	ret, err := filepath.Abs(s)
	if err != nil {
		*err1 = err.Error()
		return ""
	}
	return ret
}

func ToLowerASCII(c uint8) uint8 {
	if c >= 'A' && c <= 'Z' {
		return (c + ('a' - 'A'))
	} else {
		return c
	}
}

func EqualsCaseInsensitiveASCII(a, b string) bool {
	if len(a) != len(b) {
		return false
	}

	for i := 0; i < len(a); i++ {
		if ToLowerASCII(a[i]) != ToLowerASCII(b[i]) {
			return false
		}
	}

	return true
}

func Relativize(path string, start_list []string, err1 *string) string {
	absPath := AbsPath(path, err1)
	if *err1 != "" {
		return ""
	}

	pathList := strings.Split(absPath, "/")

	i := 0
	for i < len(start_list) && i < len(pathList) {
		if !EqualsCaseInsensitiveASCII(start_list[i], pathList[i]) {
			break
		}
		i++
	}

	relList := make([]string, 0, len(start_list)-len(pathList)+i)
	for j := i; j < len(start_list); j++ {
		relList = append(relList, "..")
	}
	for j := i; j < len(pathList); j++ {
		relList = append(relList, pathList[j])
	}

	if len(relList) == 0 {
		return "."
	}

	return strings.Join(relList, "/")
}

func IsPathSeparator(c uint8) bool {
	return c == '/' || c == '\\'
}

// Return true if paths a and b are on the same windows drive.
// Return false if this function cannot check
// whether or not on the same windows drive.
func SameDriveFast(a, b string) bool {
	if len(a) < 3 || len(b) < 3 {
		return false
	}

	if !islatinalpha(a[0]) || !islatinalpha(b[0]) {
		return false
	}

	if ToLowerASCII(a[0]) != ToLowerASCII(b[0]) {
		return false
	}

	if a[1] != ':' || b[1] != ':' {
		return false
	}

	return IsPathSeparator(a[2]) && IsPathSeparator(b[2])
}

func InternalGetFullPathName(fileName string) (string, error) {
	// 使用 filepath.Abs 获取绝对路径
	absPath, err := filepath.Abs(fileName)
	if err != nil {
		return "", fmt.Errorf("GetFullPathName(%q): %v", fileName, err)
	}
	return absPath, nil
}

// Return true if paths a and b are on the same Windows drive.
func SameDrive(a, b string, err *string) bool {
	if SameDriveFast(a, b) {
		return true
	}

	aAbsolute, err1 := InternalGetFullPathName(a)
	if err1 != nil {
		*err = err1.Error()
		return false
	}
	bAbsolute, err1 := InternalGetFullPathName(b)
	if err1 != nil {
		*err = err1.Error()
		return false
	}

	aDrive := filepath.VolumeName(aAbsolute)
	bDrive := filepath.VolumeName(bAbsolute)
	return strings.ToUpper(aDrive) == strings.ToUpper(bDrive)
}

const _MAX_PATH = 260 // max. length of full pathname

// / Normalize by fixing slashes style, fixing redundant .. and . and makes the
// / path |input| relative to |this->relative_to_| and store to |result|.
func (this *IncludesNormalize) Normalize(input *string, result1 *string, err1 *string) bool {
	if len(*input) > _MAX_PATH {
		*err1 = "path too long"
		return false
	}

	absInput := AbsPath(*input, err1)
	if *err1 != "" {
		return false
	}

	sameDrive := SameDrive(absInput, this.relative_to_, err1)
	if *err1 != "" {
		return false
	}

	if !sameDrive {
		slash_bits := uint64(0)
		copy := *input
		CanonicalizePath(&copy, &slash_bits)
		*result1 = copy
		return true
	}

	result := Relativize(absInput, this.split_relative_to_, err1)
	if err1 != nil {
		return false
	}
	*result1 = result
	return true
}
