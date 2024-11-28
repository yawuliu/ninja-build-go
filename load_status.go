package main

type LoadStatus int8

const (
	LOAD_ERROR     LoadStatus = 0
	LOAD_SUCCESS              = 1
	LOAD_NOT_FOUND            = 2
)
