package main

type Explanations interface {
	dummy()
	Record(item interface{}, fmt string, args ...interface{})
	RecordArgs(item interface{}, fmt1 string, args []interface{})
	LookupAndAppend(item interface{}, out []string)
	ptr() Explanations
}
