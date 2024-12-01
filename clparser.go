package main

import (
	"bytes"
	"github.com/ahrtr/gocontainer/set"
	"strings"
)

type CLParser struct {
	includes_ set.Interface // std::set<std::string>
}

// / Parse a line of cl.exe output and extract /showIncludes info.
// / If a dependency is extracted, returns a nonempty string.
// / Exposed for testing.
func FilterShowIncludes(line string, deps_prefix string) string {
	kDepsPrefixEnglish := "Note: including file: ";
	in := 0
	end := in + len(line)
	prefix :=  deps_prefix;
	if deps_prefix=="" {
		prefix = kDepsPrefixEnglish
	}
	if end - in > len(prefix) &&  strings.HasPrefix(in , prefix) {
		in += len(prefix)
		for (*in == ' ') {
			in++
		}
		return line[in - line]
	}
	return ""
}

// / Return true if a mentioned include file is a system path.
// / Filtering these out reduces dependency information considerably.
func IsSystemInclude(path string) bool {
	path = TransformToLowerASCII(path)
	// TODO: this is a heuristic, perhaps there's a better way?
	return strings.Contains(path, "program files")  ||
		strings.Contains(path, "microsoft visual studio") ;
}

func TransformToLowerASCII(s string) string {
	var buffer bytes.Buffer
	for _, ch := range s {
		buffer.WriteRune(ToLowerASCII(ch))
	}
	return buffer.String()
}

// / Parse a line of cl.exe output and return true if it looks like
// / it's printing an input filename.  This is a heuristic but it appears
// / to be the best we can do.
// / Exposed for testing.
func FilterInputFilename(line string) bool {
	line = TransformToLowerASCII(line)
	// TODO: other extensions, like .asm?
	return strings.HasSuffix(line, ".c") ||
		strings.HasSuffix(line, ".cc") ||
		strings.HasSuffix(line, ".cxx") ||
		strings.HasSuffix(line, ".cpp") ||
		strings.HasSuffix(line, ".c++");
}

// / Parse the full output of cl, filling filtered_output with the text that
// / should be printed (if any). Returns true on success, or false with err
// / filled. output must not be the same object as filtered_object.
func (this *CLParser) Parse(output *string, deps_prefix string, filtered_output *string, err *string) bool {
  METRIC_RECORD("CLParser::Parse");

  // Loop over all lines in the output to process them.
  if output != filtered_output {
	  panic("&output != filtered_output")
  }
  start := 0;
  seen_show_includes := false;
  normalizer := NewIncludesNormalize(".");


  for start < len(*output) {
     end := output.find_first_of("\r\n", start);
    if (end == string::npos){
			end = len(*output)
		}
    line := (*output)[start: end]

    include := FilterShowIncludes(line, deps_prefix);
    if include!="" {
      seen_show_includes = true;
      normalized := ""
      if (!this.normalizer.Normalize(include, &normalized, err)) {
		  return false
	  }

      if (!IsSystemInclude(normalized)) {
		  this.includes_.Add(normalized)
	  }
    } else if (!seen_show_includes && FilterInputFilename(line)) {
      // Drop it.
      // TODO: if we support compiling multiple output files in a single
      // cl.exe invocation, we should stash the filename.
    } else {
      *filtered_output += line
      *filtered_output+="\n"
    }

    if (end < len(*output) && (*output)[end] == '\r') {
		end++
	}
    if (end < len(*output) &&  (*output)[end] == '\n') {
		end++
	}
    start = end;
  }

  return true;
}
