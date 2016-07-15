package main

import (
	"fmt"

	"github.com/seborama/govcr"
)

const example1CassetteName = "MyCassette1"

// Example1 is an example use of govcr.
func Example1() {
	vcr := govcr.NewVCR(example1CassetteName, nil)
	resp, err := vcr.Client.Get("http://example.com/foo")
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(resp)
}
