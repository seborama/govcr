package main

import "github.com/seborama/govcr"

// Example1 is an example use of govcr.
func Example1() {
	vcr := govcr.NewVCR("MyCassette1", nil)
	vcr.Client.Get("http://example.com/foo")
}
