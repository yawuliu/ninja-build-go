package main

type ExitStatus int8

const (
	ExitSuccess     ExitStatus = 0
	ExitFailure     ExitStatus = 1
	ExitInterrupted ExitStatus = 2
)
