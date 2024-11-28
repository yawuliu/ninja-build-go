package main

import (
	"fmt"
	"os"
)

func CanonicalizePath(path *string, slash_bits *uint64) {
	len := len(*path)
	str := ""
	if len > 0 {
		str = &(*path)[0]
	}

	CanonicalizePath(str, &len, slash_bits)
	path.resize(len)
}

func Error(msg string, ap ...interface{}) {
	fmt.Fprint(os.Stderr, "ninja: error: ")
	fmt.Fprint(os.Stderr, ap...)
	fmt.Fprint(os.Stderr, "\n")
}

func Info(msg string, ap ...interface{}) {
	fmt.Fprint(os.Stdout, "ninja: ")
	fmt.Fprint(os.Stdout, ap...)
	fmt.Fprint(os.Stdout, "\n")
}

func Warning(msg string, ap ...interface{}) {
	fmt.Fprint(os.Stderr, "ninja: warning: ")
	fmt.Fprint(os.Stderr, ap...)
	fmt.Fprint(os.Stderr, "\n")
}


func SpellcheckStringV(text string,  words []string) string {
  const bool kAllowReplacements = true;
  const int kMaxValidEditDistance = 3;

  int min_distance = kMaxValidEditDistance + 1;
  const char* result = NULL;
  for (vector<const char*>::const_iterator i = words.begin();
       i != words.end(); ++i) {
    int distance = EditDistance(*i, text, kAllowReplacements,
                                kMaxValidEditDistance);
    if (distance < min_distance) {
      min_distance = distance;
      result = *i;
    }
  }
  return result;
}

func  SpellcheckString( text string, args...interface{}) string {
  // Note: This takes a const char* instead of a string& because using
  // va_start() with a reference parameter is undefined behavior.
  va_list ap;
  va_start(ap, text);
  vector<const char*> words;
  const char* word;
  while ((word = va_arg(ap, const char*)))
    words.push_back(word);
  va_end(ap);
  return SpellcheckStringV(text, words);
}

func GetWorkingDirectory() string {
 ret := ""
 success := ""
	do {
		ret.resize(ret.size() + 1024);
		errno = 0;
		success = getcwd(&ret[0], ret.size());
	} while (!success && errno == ERANGE);
	if (!success) {
		Fatal("cannot determine working directory: %s", strerror(errno));
	}
	ret.resize(strlen(&ret[0]));
	return ret
}

const EXIT_SUCCESS = 0
const EXIT_FAILURE = 1

func GetProcessorCount() int  {

}