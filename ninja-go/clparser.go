package ninja_go

import (
	"bytes"
	"strings"
)

type CLParser struct {
	includes_ map[string]bool // std::set<std::string>
}

func NewCLParser() *CLParser {
	return &CLParser{includes_: make(map[string]bool)}
}

// / Parse a line of cl.exe output and extract /showIncludes info.
// / If a dependency is extracted, returns a nonempty string.
// / Exposed for testing.
// FilterShowIncludes 过滤字符串，返回包含文件名的部分
func FilterShowIncludes(line, depsPrefix string) string {
	const kDepsPrefixEnglish = "Note: including file: "
	const trimPrefix = " "

	// 如果 depsPrefix 为空，则使用默认的前缀
	prefix := depsPrefix
	if depsPrefix == "" {
		prefix = kDepsPrefixEnglish
	}

	// 检查 line 是否以 prefix 开头
	if strings.HasPrefix(line, prefix) {
		// 找到 prefix 之后的部分
		startIndex := len(prefix)
		line = line[startIndex:]

		// 跳过前缀后的空格
		line = strings.TrimPrefix(line, trimPrefix)
		return line
	}
	return ""
}

// / Return true if a mentioned include file is a system path.
// / Filtering these out reduces dependency information considerably.
func IsSystemInclude(path string) bool {
	path = TransformToLowerASCII(path)
	// TODO: this is a heuristic, perhaps there's a better way?
	return strings.Contains(path, "program files") ||
		strings.Contains(path, "microsoft visual studio")
}

func TransformToLowerASCII(s string) string {
	var buffer bytes.Buffer
	for i := 0; i < len(s); i++ {
		ch := s[i]
		buffer.WriteByte(ToLowerASCII(ch))
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
		strings.HasSuffix(line, ".c++")
}

// findFirstOf 查找字符串 s 中第一个与子串 chars 中任一字符相匹配的字符的索引位置，从 start 开始查找
func findFirstOf(s string, chars string, start int) int {
	for i := start; i < len(s); i++ {
		if strings.ContainsRune(chars, rune(s[i])) {
			return i
		}
	}
	return -1 // 如果没有找到匹配的字符，返回 -1
}

// / Parse the full output of cl, filling filtered_output with the text that
// / should be printed (if any). Returns true on success, or false with err
// / filled. output must not be the same object as filtered_object.
func (this *CLParser) Parse(output *string, deps_prefix string, filtered_output *string, err *string) bool {
	METRIC_RECORD("CLParser::Parse")

	// Loop over all lines in the output to process them.
	if output == filtered_output {
		panic("&output == filtered_output")
	}
	start := 0
	seen_show_includes := false
	normalizer := NewIncludesNormalize(".")

	for start < len(*output) {
		end := findFirstOf(*output, "\r\n", start)
		if end == -1 {
			end = len(*output)
		}
		line := (*output)[start:end]

		include := FilterShowIncludes(line, deps_prefix)
		if include != "" {
			seen_show_includes = true
			normalized := ""
			if !normalizer.Normalize(&include, &normalized, err) {
				return false
			}

			if !IsSystemInclude(normalized) {
				this.includes_[normalized] = true
			}
		} else if !seen_show_includes && FilterInputFilename(line) {
			// Drop it.
			// TODO: if we support compiling multiple output files in a single
			// cl.exe invocation, we should stash the filename.
		} else {
			*filtered_output += line
			*filtered_output += "\n"
		}

		if end < len(*output) && (*output)[end] == '\r' {
			end++
		}
		if end < len(*output) && (*output)[end] == '\n' {
			end++
		}
		start = end
	}

	return true
}
