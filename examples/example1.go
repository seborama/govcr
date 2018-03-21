package main

import (
	"fmt"

	"github.com/seborama/govcr"
)

const example1CassetteName = "MyCassette1"

// Example1 is an example use of govcr.
func Example1() {
	vcr := govcr.NewVCR(example1CassetteName, nil)
	vcr.Client.Get("http://www.example.com/foo")
	fmt.Printf("%+v\n", vcr.Stats())
}
