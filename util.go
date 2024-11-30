package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

func islatinalpha(c int) bool {
	// isalpha() is locale-dependent.
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}

func StripAnsiEscapeCodes(in string) string {
	stripped := ""
	// stripped.reserve(in.size());

	for i := 0; i < len(in); i++ {
		if in[i] != '\033' {
			// Not an escape code.
			stripped += string(in[i])
			continue
		}

		// Only strip CSIs for now.
		if i+1 >= len(in) {
			break
		}
		if in[i+1] != '[' {
			continue
		} // Not a CSI.
		i += 2

		// Skip everything up to and including the next [a-zA-Z].
		for i < len(in) && !islatinalpha(int(in[i])) {
			i++
		}
	}
	return stripped
}

func CanonicalizePath(path *string, slash_bits *uint64) {
	// 确保路径是绝对的
	absPath, err := filepath.Abs(*path)
	if err != nil {
		fmt.Printf("Error getting absolute path: %v\n", err)
		return
	}
	// 清理路径，去掉多余的分隔符和".."
	cleanPath := filepath.Clean(absPath)

	// 如果需要，这里可以进一步处理路径，例如转换所有分隔符为一个特定的字符
	// 但在Go中，这通常不是必需的，因为filepath.Clean已经处理了路径分隔符

	// 更新传入的路径指针
	*path = cleanPath
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

// SpellcheckStringV 接受一个字符串和一个字符串切片，返回编辑距离最小的字符串。
func SpellcheckStringV(text string, words []string) string {
	const kAllowReplacements = true
	const kMaxValidEditDistance = 3

	var minDistance = kMaxValidEditDistance + 1
	var result string
	for _, word := range words {
		distance := EditDistance(word, text, kAllowReplacements, kMaxValidEditDistance)
		if distance < minDistance {
			minDistance = distance
			result = word
		}
	}
	return result
}

func SpellcheckString(text string, words ...string) string {
	// Note: This takes a const char* instead of a string& because using
	// va_start() with a reference parameter is undefined behavior.
	return SpellcheckStringV(text, words)
}

func GetWorkingDirectory() string {
	currentDir, err := os.Getwd()
	if err != nil {
		log.Fatalf("cannot determine working directory: %v", err)
	}
	return currentDir
}

const EXIT_SUCCESS = 0
const EXIT_FAILURE = 1

func GetProcessorCount() int {
	return runtime.NumCPU()
}

// StringNeedsWin32Escaping 检查字符串是否需要转义
func StringNeedsWin32Escaping(input string) bool {
	for _, c := range input {
		if c == '"' || c == '\\' {
			return true
		}
	}
	return false
}

// GetWin32EscapedString 转义字符串以在Windows命令行中使用
func GetWin32EscapedString(input string, result1 *string) {
	if !StringNeedsWin32Escaping(input) {
		*result1 = input
		return
	}

	var result strings.Builder
	const kQuote = '"'
	const kBackslash = '\\'
	result.WriteRune(kQuote)

	var consecutiveBackslashCount int
	spanBegin := 0

	for i, c := range input {
		switch c {
		case kBackslash:
			consecutiveBackslashCount++
		case kQuote:
			result.WriteString(input[spanBegin:i])
			result.WriteString(strings.Repeat(string(kBackslash), consecutiveBackslashCount+1))
			spanBegin = i
			consecutiveBackslashCount = 0
		default:
			consecutiveBackslashCount = 0
		}
	}
	result.WriteString(input[spanBegin:])
	result.WriteString(strings.Repeat(string(kBackslash), consecutiveBackslashCount))
	result.WriteRune(kQuote)
	*result1 = result.String()
}
