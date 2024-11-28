package main

type DirCache map[string]TimeStamp
type Cache map[string]DirCache
type RealDiskInterface struct {
	DiskInterface
	/// Whether stat information can be cached.
	use_cache_ bool

	/// Whether long paths are enabled.
	long_paths_enabled_ bool

	// TODO: Neither a map nor a hashmap seems ideal here.  If the statcache
	// works out, come up with a better data structure.
	cache_ Cache
}

type FileReader interface {
	ReadFile(path string, contents *string, err *string) Status
}

type DiskInterface interface {
	FileReader
	Stat(path string, err *string) TimeStamp
	WriteFile(path string, contents string) bool
	MakeDir(path string) bool
	ReadFile(path string, contents, err *string) Status
	RemoveFile(path string) int
}
