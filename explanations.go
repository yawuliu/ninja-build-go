package main

import "fmt"

type OptionalExplanations struct {
	explanations_ *Explanations
}

func NewOptionalExplanations(explanations *Explanations) *OptionalExplanations {
	ret := OptionalExplanations{}
	ret.explanations_ = explanations
	return &ret
}

func (this *OptionalExplanations) Record(item interface{}, fmt string, args ...interface{}) {
	if this.explanations_ != nil {
		this.explanations_.RecordArgs(item, fmt, args)
	}
}

func (this *OptionalExplanations) RecordArgs(item interface{}, fmt string, args []interface{}) {
	if this.explanations_ != nil {
		this.explanations_.RecordArgs(item, fmt, args)
	}
}

func (this *OptionalExplanations) LookupAndAppend(item interface{}, out []string) {
	if this.explanations_ != nil {
		this.explanations_.LookupAndAppend(item, out)
	}
}

func (this *OptionalExplanations) ptr() *Explanations { return this.explanations_ }

func (this *Explanations) Record(item interface{}, fmt string, args ...interface{}) {
	this.RecordArgs(item, fmt, args)
}

// / Same as Record(), but uses a va_list to pass formatting arguments.
func (this *Explanations) RecordArgs(item interface{}, fmt1 string, args []interface{}) {
	buffer := ""
	fmt.Sprintf(buffer, fmt1, args)
	this.map_[item] = append(this.map_[item], buffer)
}

// / Lookup the explanations recorded for |item|, and append them
// / to |*out|, if any.
func (this *Explanations) LookupAndAppend(item interface{}, out []string) {
	it, ok := this.map_[item]
	if !ok {
		return
	}

	for _, explanation := range it {
		out = append(out, explanation)
	}
}
