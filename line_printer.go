package main

import (
	"fmt"
	"os"
	"syscall"
)

type LineType int8

const (
	FULL  LineType = 0
	ELIDE LineType = 1
)

type LinePrinter struct {
	/// Whether we can do fancy terminal control codes.
	smart_terminal_ bool

	/// Whether we can use ISO 6429 (ANSI) color sequences.
	supports_color_ bool

	/// Whether the caret is at the beginning of a blank line.
	have_blank_line_ bool

	/// Whether console is locked.
	console_locked_ bool

	/// Buffered current line while console is locked.
	line_buffer_ string

	/// Buffered line type while console is locked.
	line_type_ LineType

	/// Buffered console output while console is locked.
	output_buffer_ string

	console_ interface{}
}

// IsTty 检查给定的文件描述符是否指向一个终端
func isatty(fd int) bool {
	Stat, err := os.Stat(fmt.Sprintf("/proc/self/fd/%d", fd))
	if err != nil {
		return false
	}
	return Stat.Mode()&syscall.S_IFMT == syscall.S_IFCHR
}

func NewLinePrinter() *LinePrinter {
	ret := LinePrinter{}
	ret.have_blank_line_ = true
	ret.console_locked_ = false
	term := os.Getenv("TERM")

	ret.smart_terminal_ = isatty(1) && term != "" && string(term) != "dumb"
	ret.supports_color_ = ret.smart_terminal_

	// Try enabling ANSI escape sequence support on Windows 10 terminals.
	//if ret.supports_color_ {
	//  DWORD mode;
	//  if (GetConsoleMode(console_, &mode)) {
	//    if (!SetConsoleMode(console_, mode | ENABLE_VIRTUAL_TERMINAL_PROCESSING)) {
	//	  ret.supports_color_ = false;
	//    }
	//  }
	//}

	if !ret.supports_color_ {
		clicolor_force := os.Getenv("CLICOLOR_FORCE")
		ret.supports_color_ = clicolor_force != "" && clicolor_force != "0"
	}
	return &ret
}
func (this *LinePrinter) is_smart_terminal() bool       { return this.smart_terminal_ }
func (this *LinePrinter) set_smart_terminal(smart bool) { this.smart_terminal_ = smart }

func (this *LinePrinter) supports_color() bool { return this.supports_color_ }

// getTerminalWidth 获取终端的宽度
func getTerminalWidth() (int, error) {
	return 800, nil
}

// elideMiddleInPlace 截断字符串以适应给定的宽度
func elideMiddleInPlace(s string, maxWidth int) string {
	if len(s) <= maxWidth {
		return s
	}
	half := (maxWidth - 3) / 2 // 保留 "..." 的空间
	return s[:half] + "..." + s[len(s)-half:]
}

// printString 打印字符串，如果终端宽度有限，则截断中间部分
func (this *LinePrinter) PrintString(toPrint string) {
	width, err := getTerminalWidth()
	if err == nil && width > 0 {
		toPrint = elideMiddleInPlace(toPrint, width)
	}
	fmt.Print(toPrint)  // \r 用于覆盖上一行
	fmt.Print("\033[K") // 清除行尾
}

// / Overprints the current line. If type is ELIDE, elides to_print to fit on
// / one line.
func (this *LinePrinter) Print(to_print string, lineType LineType) {
	if this.console_locked_ {
		this.line_buffer_ = to_print
		this.line_type_ = lineType
		return
	}

	if this.smart_terminal_ {
		fmt.Print("\r") // 覆盖上一行
	}

	if this.smart_terminal_ && lineType == ELIDE {
		this.PrintString(to_print)
	} else {
		fmt.Printf("%s\n", to_print)
	}
}

// / Prints a string on a new line, not overprinting previous output.
func (this *LinePrinter) PrintOnNewLine(to_print string) {
	if this.console_locked_ && this.line_buffer_ != "" {
		this.output_buffer_ += this.line_buffer_
		this.output_buffer_ += string('\n')
		this.line_buffer_ = ""
	}
	if !this.have_blank_line_ {
		this.PrintOrBuffer("\n", 1)
	}
	if to_print != "" {
		this.PrintOrBuffer(to_print, int64(len(to_print)))
	}
	this.have_blank_line_ = to_print == "" || to_print[len(to_print)-1] == '\n'
}

// / Lock or unlock the console.  Any output sent to the LinePrinter while the
// / console is locked will not be printed until it is unlocked.
func (this *LinePrinter) SetConsoleLocked(locked bool) {
	if locked == this.console_locked_ {
		return
	}

	if locked {
		this.PrintOnNewLine("")
	}

	this.console_locked_ = locked

	if !locked {
		this.PrintOnNewLine(this.output_buffer_)
		if this.line_buffer_ != "" {
			this.Print(this.line_buffer_, this.line_type_)
		}
		this.output_buffer_ = ""
		this.line_buffer_ = ""
	}
}

// / Print the given data to the console, or buffer it if it is locked.
func (this *LinePrinter) PrintOrBuffer(data string, size int64) {
	if this.console_locked_ {
		this.output_buffer_ += data
	} else {
		// Avoid printf and C strings, since the actual output might contain null
		// bytes like UTF-16 does (yuck).
		fmt.Fprintf(os.Stdout, data)
	}
}
