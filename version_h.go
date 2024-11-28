package main

import (
	"log"
	"strconv"
	"strings"
)

// / The version number of the current Ninja release.  This will always
// / be "git" on trunk.
const kNinjaVersion = "1.13.0.git"

// ParseVersion 解析版本字符串，提取主版本号和次版本号。
func ParseVersion(version string, major, minor *int) {
	parts := strings.Split(version, ".")
	if len(parts) > 0 {
		*major, _ = strconv.Atoi(parts[0])
	}
	*minor = 0
	if len(parts) > 1 {
		*minor, _ = strconv.Atoi(parts[1])
	}
}

func CheckNinjaVersion(version string) {
	bin_major := 0
	bin_minor := 0
	ParseVersion(kNinjaVersion, &bin_major, &bin_minor)
	file_major := 0
	file_minor := 0
	ParseVersion(version, &file_major, &file_minor)

	if bin_major > file_major {
		Warning("ninja executable version (%s) greater than build file "+
			"ninja_required_version (%s); versions may be incompatible.",
			kNinjaVersion, version)
		return
	}

	if (bin_major == file_major && bin_minor < file_minor) ||
		bin_major < file_major {
		log.Fatalf("ninja version (%s) incompatible with build file "+
			"ninja_required_version version (%s).",
			kNinjaVersion, version)
	}
}
