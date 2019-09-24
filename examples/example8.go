package main

import (
	"fmt"

	"github.com/seborama/govcr"
)

const example8CassetteName = "MyCassette8"

// Example8 is an example use of SkipErrorCodes feature.
func Example8() {
	cfg := govcr.VCRConfig{
		Logging: true,
		SkipErrorCodes: true,
	}
	vcr := govcr.NewVCR(example1CassetteName, &cfg)
	vcr.Client.Get("http://www.example.com/foo")
	fmt.Printf("%+v\n", vcr.Stats())
}
