package ninja_go

import (
	"bytes"
	"strings"
)

type DepfileParserOptions struct {
}

func NewDepfileParserOptions() *DepfileParserOptions {
	ret := DepfileParserOptions{}
	return &ret
}

type DepfileParser struct {
	outs_    []*string
	ins_     []*string
	options_ *DepfileParserOptions
}

func NewDepfileParser(options *DepfileParserOptions) *DepfileParser {
	ret := DepfileParser{}
	ret.options_ = options
	return &ret
}

// / Parse an input file.  Input must be NUL-terminated.
// / Warning: may mutate the content in-place and parsed StringPieces are
// / pointers within it.
func (this *DepfileParser) Parse(content string, err *string) bool {
	this.outs_ = this.outs_[:0] // 重用切片
	this.ins_ = this.ins_[:0]   // 重用切片
	*err = ""

	var out bytes.Buffer
	var parsingTargets bool = true
	var poisonedInput bool = false

	lines := strings.Split(content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if len(line) == 0 {
			continue
		}

		isDependency := !parsingTargets
		filename := line
		if parsingTargets && strings.HasSuffix(line, ":") {
			filename = line[:len(line)-1] // 移除行尾的冒号
			parsingTargets = false
		}

		if len(filename) > 0 {
			// 处理反斜杠转义字符
			var unescaped bytes.Buffer
			isEscaped := false
			for i := 0; i < len(filename); i++ {
				if isEscaped {
					unescaped.WriteByte(filename[i])
					isEscaped = false
				} else if filename[i] == '\\' {
					isEscaped = true
				} else {
					unescaped.WriteByte(filename[i])
				}
			}
			filename = unescaped.String()

			// 检查是否已经出现过
			if _, exists := this.findString(this.outs_, filename); isDependency && exists {
				*err = "inputs may not also have inputs"
				return false
			}

			if isDependency {
				if poisonedInput {
					*err = "inputs may not follow outputs"
					return false
				}
				this.ins_ = append(this.ins_, &filename)
			} else {
				if _, found := this.findString(this.ins_, filename); !found {
					this.outs_ = append(this.outs_, &filename)
				}
			}
		}

		if strings.HasSuffix(line, "\\") {
			out.WriteString(line[:len(line)-1] + "\n")
		} else {
			out.WriteString(line + "\n")
			parsingTargets = true
			poisonedInput = false
		}
	}

	if len(this.outs_) == 0 {
		*err = "expected ':' in depfile"
		return false
	}

	return true
}

// FindString 在切片中查找字符串，如果找到返回索引和true，否则返回-1和false
func (p *DepfileParser) findString(slice []*string, str string) (int, bool) {
	for i, s := range slice {
		if *s == str {
			return i, true
		}
	}
	return -1, false
}
