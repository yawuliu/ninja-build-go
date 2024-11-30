package main

type DepfileParser struct {
	outs_    []string
	ins_     []string
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
func (this *DepfileParser) Parse(content, err *string) bool {
  // in: current parser input point.
  // end: end of input.
  // parsing_targets: whether we are parsing targets or dependencies.
  char* in = &(*content)[0];
  char* end = in + content.size();
  have_target := false;
  parsing_targets := true;
  poisoned_input := false;
  is_empty := true;
  while (in < end) {
    have_newline := false;
    // out: current output point (typically same as in, but can fall behind
    // as we de-escape backslashes).
    char* out = in;
    // filename: start of the current parsed filename.
    char* filename = out;
    for  {
      // start: beginning of the current parsed span.
      const char* start = in;
      char* yymarker = NULL;
      
    {
      unsigned char yych;
      yybm := []uint8{
          0,   0,   0,   0,   0,   0,   0,   0, 
          0,   0,   0,   0,   0,   0,   0,   0, 
          0,   0,   0,   0,   0,   0,   0,   0, 
          0,   0,   0,   0,   0,   0,   0,   0, 
          0, 128,   0,   0,   0, 128,   0,   0, 
        128, 128,   0, 128, 128, 128, 128, 128, 
        128, 128, 128, 128, 128, 128, 128, 128, 
        128, 128, 128,   0,   0, 128,   0,   0, 
        128, 128, 128, 128, 128, 128, 128, 128, 
        128, 128, 128, 128, 128, 128, 128, 128, 
        128, 128, 128, 128, 128, 128, 128, 128, 
        128, 128, 128, 128,   0, 128,   0, 128, 
          0, 128, 128, 128, 128, 128, 128, 128, 
        128, 128, 128, 128, 128, 128, 128, 128, 
        128, 128, 128, 128, 128, 128, 128, 128, 
        128, 128, 128, 128,   0, 128, 128,   0, 
        128, 128, 128, 128, 128, 128, 128, 128, 
        128, 128, 128, 128, 128, 128, 128, 128, 
        128, 128, 128, 128, 128, 128, 128, 128, 
        128, 128, 128, 128, 128, 128, 128, 128, 
        128, 128, 128, 128, 128, 128, 128, 128, 
        128, 128, 128, 128, 128, 128, 128, 128, 
        128, 128, 128, 128, 128, 128, 128, 128, 
        128, 128, 128, 128, 128, 128, 128, 128, 
        128, 128, 128, 128, 128, 128, 128, 128, 
        128, 128, 128, 128, 128, 128, 128, 128, 
        128, 128, 128, 128, 128, 128, 128, 128, 
        128, 128, 128, 128, 128, 128, 128, 128, 
        128, 128, 128, 128, 128, 128, 128, 128, 
        128, 128, 128, 128, 128, 128, 128, 128, 
        128, 128, 128, 128, 128, 128, 128, 128, 
        128, 128, 128, 128, 128, 128, 128, 128, 
      };
      yych = *in;
      if (yybm[0+yych] & 128) {
        goto yy9;
      }
      if (yych <= '\r') {
        if (yych <= '\t') {
          if (yych >= 0x01) {
			  goto yy4
		  }
        } else {
          if (yych <= '\n') {
			  goto yy6
		  }
          if (yych <= '\f') {
			  goto yy4
		  }
          goto yy8;
        }
      } else {
        if (yych <= '$') {
          if (yych <= '#') {
			  goto yy4
		  }
          goto yy12;
        } else {
          if (yych <= '?') {
			  goto yy4
		  }
          if (yych <= '\\') {
			  goto yy13
		  }
          goto yy4;
        }
      }
      in++
      {
        break;
      }
yy4:
      in++
yy5:
      {
        // For any other character (e.g. whitespace), swallow it here,
        // allowing the outer logic to loop around again.
        break;
      }
yy6:
      in++
      {
        // A newline ends the current file name and the current rule.
        have_newline = true;
        break;
      }
yy8:
      yych = *++in;
      if (yych == '\n') {
		  goto yy6
	  }
      goto yy5;
yy9:
      yych = *++in;
      if (yybm[0+yych] & 128) {
        goto yy9;
      }
yy11:
      {
        // Got a span of plain text.
        len := int(in - start);
        // Need to shift it over if we're overwriting backslashes.
        if (out < start) {
			memmove(out, start, len)
		}
        out += len;
        continue;
      }
yy12:
      yych = *++in;
      if (yych == '$') {
		  goto yy14
	  }
      goto yy5;
yy13:
      yych = *(yymarker = ++in);
      if (yych <= ' ') {
        if (yych <= '\n') {
          if (yych <= 0x00) {
			  goto yy5
		  }
          if (yych <= '\t') {
			  goto yy16
		  }
          goto yy17;
        } else {
          if (yych == '\r') {
			  goto yy19
		  }
          if (yych <= 0x1F) {
			  goto yy16
		  }
          goto yy21;
        }
      } else {
        if (yych <= '9') {
          if (yych == '#') {
			  goto yy23
		  }
          goto yy16;
        } else {
          if (yych <= ':') {
			  goto yy25
		  }
          if (yych == '\\') {
			  goto yy27
		  }
          goto yy16;
        }
      }
yy14:
      in++
      {
        // De-escape dollar character.
        *out++ = '$';
        continue;
      }
yy16:
      in++
      goto yy11;
yy17:
      in++
      {
        // A line continuation ends the current file name.
        break;
      }
yy19:
      yych = *++in;
      if (yych == '\n') {
		  goto yy17
	  }
      in = yymarker;
      goto yy5;
yy21:
      in++
      {
        // 2N+1 backslashes plus space . N backslashes plus space.
        len := int(in - start);
        n := len / 2 - 1;
        if (out < start) {
			memset(out, '\\', n)
		}
        out += n;
        *out++ = ' ';
        continue;
      }
yy23:
      in++
      {
        // De-escape hash sign, but preserve other leading backslashes.
         len := int(in - start);
        if (len > 2 && out < start) {
			memset(out, '\\', len-2)
		}
        out += len - 2;
        *out++ = '#';
        continue;
      }
yy25:
      yych = *++in;
      if (yych <= '\f') {
        if (yych <= 0x00) {
			goto yy28
		}
        if (yych <= 0x08) {
			goto yy26
		}
        if (yych <= '\n') {
			goto yy28
		}
      } else {
        if (yych <= '\r') {
			goto yy28
		}
        if (yych == ' ') {
			goto yy28
		}
      }
yy26:
      {
        // De-escape colon sign, but preserve other leading backslashes.
        // Regular expression uses lookahead to make sure that no whitespace
        // nor EOF follows. In that case it'd be the : at the end of a target
        len := int(in - start);
        if (len > 2 && out < start) {
			memset(out, '\\', len-2)
		}
        out += len - 2;
        *out++ = ':';
        continue;
      }
yy27:
      yych = *++in;
      if (yych <= ' ') {
        if (yych <= '\n') {
          if (yych <= 0x00) {
			  goto yy11
		  }
          if (yych <= '\t') {
			  goto yy16
		  }
          goto yy11;
        } else {
          if (yych == '\r') {
			  goto yy11
		  }
          if (yych <= 0x1F) {
			  goto yy16
		  }
          goto yy30;
        }
      } else {
        if (yych <= '9') {
          if (yych == '#') {
			  goto yy23
		  }
          goto yy16;
        } else {
          if (yych <= ':') {
			  goto yy25
		  }
          if (yych == '\\') {
			  goto yy32
		  }
          goto yy16;
        }
      }
yy28:
      in++
      {
        // Backslash followed by : and whitespace.
        // It is therefore normal text and not an escaped colon
        len := (int)(in - start - 1);
        // Need to shift it over if we're overwriting backslashes.
        if (out < start) {
			memmove(out, start, len)
		}
        out += len;
        if (*(in - 1) == '\n') {
			have_newline = true
		}
        break;
      }
yy30:
      in++
      {
        // 2N backslashes plus space . 2N backslashes, end of filename.
        len := int(in - start);
        if (out < start) {
			memset(out, '\\', len-1)
		}
        out += len - 1;
        break;
      }
yy32:
      yych = *++in;
      if (yych <= ' ') {
        if (yych <= '\n') {
          if (yych <= 0x00) {
			  goto yy11
		  }
          if (yych <= '\t') {
			  goto yy16
		  }
          goto yy11;
        } else {
          if (yych == '\r') {
			  goto yy11
		  }
          if (yych <= 0x1F) {
			  goto yy16
		  }
          goto yy21;
        }
      } else {
        if (yych <= '9') {
          if (yych == '#') {
			  goto yy23
		  }
          goto yy16;
        } else {
          if (yych <= ':') {
			  goto yy25
		  }
          if (yych == '\\') {
			  goto yy27
		  }
          goto yy16;
        }
      }
    }

    }

    len := int(out - filename);
     is_dependency := !parsing_targets;
    if (len > 0 && filename[len - 1] == ':') {
      len--;  // Strip off trailing colon, if any.
      parsing_targets = false;
      have_target = true;
    }

    if (len > 0) {
      is_empty = false;
       piece := string(filename, len);
      // If we've seen this as an input before, skip it.
      pos := std::find(this.ins_.begin(), this.ins_.end(), piece);
      if (pos == this.ins_.end()) {
        if (is_dependency) {
          if (poisoned_input) {
            *err = "inputs may not also have inputs";
            return false;
          }
          // New input.
			this.ins_ = append(this.ins_ , piece)
        } else {
          // Check for a new output.
          if (std::find(this.outs_.begin(), this.outs_.end(), piece) == this.outs_.end()){
				this.outs_ = append(this.outs_ , piece)
		  }
        }
      } else if (!is_dependency) {
        // We've passed an input on the left side; reject new inputs.
        poisoned_input = true;
      }
    }

    if (have_newline) {
      // A newline ends a rule so the next filename will be a new target.
      parsing_targets = true;
      poisoned_input = false;
    }
  }
  if (!have_target && !is_empty) {
    *err = "expected ':' in depfile";
    return false;
  }
  return true;
}
