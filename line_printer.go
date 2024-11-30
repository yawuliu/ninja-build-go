package main

import (
	"fmt"
	"os"
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

func NewLinePrinter() *LinePrinter                      {
	ret := LinePrinter{}
	ret.have_blank_line_ = true
	ret.console_locked_ = false
 term := os.Getenv("TERM");

	ret.smart_terminal_ = isatty(1) && term && string(term) != "dumb";
	ret.supports_color_ = ret.smart_terminal_;

  // Try enabling ANSI escape sequence support on Windows 10 terminals.
  if ret.supports_color_ {
    DWORD mode;
    if (GetConsoleMode(console_, &mode)) {
      if (!SetConsoleMode(console_, mode | ENABLE_VIRTUAL_TERMINAL_PROCESSING)) {
		  ret.supports_color_ = false;
      }
    }
  }

  if (!ret.supports_color_) {
    clicolor_force := os.Getenv("CLICOLOR_FORCE");
	  ret.supports_color_ = clicolor_force && std::string(clicolor_force) != "0";
  }
  return &ret
}
func (this *LinePrinter) is_smart_terminal() bool       { return this.smart_terminal_ }
func (this *LinePrinter) set_smart_terminal(smart bool) { this.smart_terminal_ = smart }

func (this *LinePrinter) supports_color() bool { return this.supports_color_ }

// / Overprints the current line. If type is ELIDE, elides to_print to fit on
// / one line.
func (this *LinePrinter) Print(to_print string, tp LineType) {
 if this.console_locked_ {
	 this.line_buffer_ = to_print;
	 this.line_type_ = tp
    return;
  }

  if (this.smart_terminal_) {
    printf("\r");  // Print over previous line, if any.
    // On Windows, calling a C library function writing to stdout also handles
    // pausing the executable when the "Pause" key or Ctrl-S is pressed.
  }

  if (this.smart_terminal_ && type == ELIDE) {
    CONSOLE_SCREEN_BUFFER_INFO csbi;
    GetConsoleScreenBufferInfo(console_, &csbi);

    ElideMiddleInPlace(to_print, static_cast<size_t>(csbi.dwSize.X));
    if (this.supports_color_) {  // this means ENABLE_VIRTUAL_TERMINAL_PROCESSING
                            // succeeded
      printf("%s\x1B[K", to_print.c_str());  // Clear to end of line.
      fflush(stdout);
    } else {
      // We don't want to have the cursor spamming back and forth, so instead of
      // printf use WriteConsoleOutput which updates the contents of the buffer,
      // but doesn't move the cursor position.
      COORD buf_size = { csbi.dwSize.X, 1 };
      COORD zero_zero = { 0, 0 };
      SMALL_RECT target = { csbi.dwCursorPosition.X, csbi.dwCursorPosition.Y,
                            static_cast<SHORT>(csbi.dwCursorPosition.X +
                                               csbi.dwSize.X - 1),
                            csbi.dwCursorPosition.Y };
      vector<CHAR_INFO> char_data(csbi.dwSize.X);
      for (size_t i = 0; i < static_cast<size_t>(csbi.dwSize.X); ++i) {
        char_data[i].Char.AsciiChar = i < to_print.size() ? to_print[i] : ' ';
        char_data[i].Attributes = csbi.wAttributes;
      }
      WriteConsoleOutput(console_, &char_data[0], buf_size, zero_zero, &target);
    }
    this.have_blank_line_ = false;
  } else {
    printf("%s\n", to_print.c_str());
    fflush(stdout);
  }
}

// / Prints a string on a new line, not overprinting previous output.
func (this *LinePrinter) PrintOnNewLine(to_print string) {
	if this.console_locked_ && this.line_buffer_ != "" {
		this.output_buffer_ += this.line_buffer_
		this.output_buffer_ += '\n'
		this.line_buffer_ = ""
	}
	if !this.have_blank_line_ {
		this.PrintOrBuffer("\n", 1)
	}
	if to_print != "" {
		this.PrintOrBuffer(to_print, len(to_print))
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
