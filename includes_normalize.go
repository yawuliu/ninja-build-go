package main

import (
	"log"
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
func AbsPath(s string, err *string) string {
	if IsFullPathName(s) {
		result := s
		for i := 0; i < len(result); i++ {
			if result[i] == '\\' {
				result[i] = '/'
			}
		}
		return result
	}

	result := ""
	if !InternalGetFullPathName(s, result, sizeof(result), err) {
		return ""
	}
	for c := result; *c; c++ {
		if *c == '\\' {
			*c = '/'
		}
	}
	return result
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

// Return true if paths a and b are on the same windows drive.
// Return false if this function cannot check
// whether or not on the same windows drive.
func SameDriveFast(a, b string) bool {
	if a.size() < 3 || b.size() < 3 {
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

// Return true if paths a and b are on the same Windows drive.
func SameDrive(a, b string, err *string) bool {
	if SameDriveFast(a, b) {
		return true
	}

	a_absolute := ""
	b_absolute := ""
	if !InternalGetFullPathName(a, a_absolute, sizeof(a_absolute), err) {
		return false
	}
	if !InternalGetFullPathName(b, b_absolute, sizeof(b_absolute), err) {
		return false
	}
	a_drive := ""
	b_drive := ""
	_splitpath(a_absolute, a_drive, NULL, NULL, NULL)
	_splitpath(b_absolute, b_drive, NULL, NULL, NULL)
	return _stricmp(a_drive, b_drive) == 0
}

const _MAX_PATH = 260 // max. length of full pathname

// / Normalize by fixing slashes style, fixing redundant .. and . and makes the
// / path |input| relative to |this->relative_to_| and store to |result|.
func (this *IncludesNormalize) Normalize(input string, result1 *string, err1 *string) bool {
	if len(input) > _MAX_PATH {
		*err1 = "path too long"
		return false
	}

	absInput := AbsPath(input, err1)
	if *err1 != "" {
		return false
	}

	sameDrive, err := SameDrive(absInput, this.relative_to_)
	if err1 != "" {
		return false
	}

	if !sameDrive {
		partiallyFixed := CanonicalizePath(&input)
		return partiallyFixed
	}

	result := Relativize(absInput, this.split_relative_to_, err1)
	if err1 != nil {
		return false
	}
	*result1 = result
	return true
}
