package main

import "fmt"

type Explanations interface {
	dummy()
	Record(item interface{}, fmt string, args ...interface{})
	RecordArgs(item interface{}, fmt1 string, args []interface{})
	LookupAndAppend(item interface{}, out []string)
	ptr() Explanations
}

type OptionalExplanations struct { // explanations_
	Explanations
	map_ map[interface{}][]string
}

func (this *OptionalExplanations) dummy() {}

func NewOptionalExplanations() *OptionalExplanations { //explanations Explanations
	ret := OptionalExplanations{}
	ret.map_ = make(map[interface{}][]string)
	// ret.explanations_ = explanations
	return &ret
}

func (this *OptionalExplanations) Record(item interface{}, fmt string, args ...interface{}) {
	this.RecordArgs(item, fmt, args)
}

// / Same as Record(), but uses a va_list to pass formatting arguments.
func (this *OptionalExplanations) RecordArgs(item interface{}, fmt1 string, args []interface{}) {
	buffer := ""
	fmt.Sprintf(buffer, fmt1, args)
	this.map_[item] = append(this.map_[item], buffer)
}

// / Lookup the explanations recorded for |item|, and append them
// / to |*out|, if any.
func (this *OptionalExplanations) LookupAndAppend(item interface{}, out []string) {
	it, ok := this.map_[item]
	if !ok {
		return
	}

	for _, explanation := range it {
		out = append(out, explanation)
	}
}

func (this *OptionalExplanations) ptr() Explanations { return this }
