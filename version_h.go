package main

// / The version number of the current Ninja release.  This will always
// / be "git" on trunk.
const kNinjaVersion = "1.13.0.git"

func ParseVersion(version string, major, minor *int) {
	end := version.find('.');
	*major = atoi(version.substr(0, end))
	*minor = 0;
	if (end != string::npos) {
		start := end + 1;
		end = version.find('.', start);
		*minor = atoi(version.substr(start, end))
	}
}

func  CheckNinjaVersion(version string) {
	bin_major := 0
	bin_minor:= 0
	ParseVersion(kNinjaVersion, &bin_major, &bin_minor)
	file_major := 0
	file_minor  := 0
	ParseVersion(version, &file_major, &file_minor)

	if (bin_major > file_major) {
		Warning("ninja executable version (%s) greater than build file "
		"ninja_required_version (%s); versions may be incompatible.",
			kNinjaVersion, version)
		return;
	}

	if ((bin_major == file_major && bin_minor < file_minor) ||
		bin_major < file_major) {
		Fatal("ninja version (%s) incompatible with build file "
		"ninja_required_version version (%s).",
			kNinjaVersion, version)
	}
}
