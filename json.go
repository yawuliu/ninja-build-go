package main

import (
	"encoding/json"
	"fmt"
	"os"
)

func PrintJSONString(in string) {
	var val interface{}
	err := json.Unmarshal([]byte(in), &val)
	if err != nil {
		panic(err)
	}
	out, err := json.MarshalIndent(val, "", "  ")
	if err != nil {
		fmt.Println("JSON encoding failed:", err)
		return
	}
	fmt.Fprintf(os.Stdout, string(out))
}
