package main

type StringPiece struct {
	str_ string
	len_ int64
}

func NewStringPiece() *StringPiece {
	ret := StringPiece{}
	ret.str_ = ""
	ret.len_ = 0
	return &ret
}

/// The constructors intentionally allow for implicit conversions.
func NewStringPiece0(str string) *StringPiece {
	str_(str.data()), len_(str.size())
}
func NewStringPiece1(str string) *StringPiece {
	str_(str), len_(strlen(str))
}

func NewStringPiece2(str string, len int64) *StringPiece {
	str_(str), len_(len)
}

func (this *StringPiece) CompareStringPieceEq(other *StringPiece) bool {
	return this.len_ == other.len_ && memcmp(this.str_, other.str_, this.len_) == 0
}

func (this *StringPiece) CompareStringPieceNe(other *StringPiece) bool {
	return !(*this == other)
}

/// Convert the slice into a full-fledged std::string, copying the
/// data into a new string.
func (this *StringPiece) AsString() string {
	if this.len_ > 0 {
		return this.str_
	} else {
		return ""
	}
}

func (this *StringPiece) begin() int {
	return this.str_
}

func (this *StringPiece) end() int {
	return this.str_ + this.len_
}

func (this *StringPiece) At(pos int64) int {
	return this.str_[pos]
}

func (this *StringPiece) size() int64 {
	return this.len_
}

func (this *StringPiece) empty() bool {
	return this.len_ == 0
}
