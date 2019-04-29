package govcr

import (
	"log"
	"net/http"
)

type Setting func(vcrConfig *VCRSettings)

func WithClient(httpClient *http.Client) Setting {
	return func(vcrConfig *VCRSettings) {
		vcrConfig.client = httpClient
	}
}

func WithCassette(cassetteName string) Setting {
	return func(vcrConfig *VCRSettings) {
		k7, err := loadCassette(cassetteName)
		if err != nil {
			log.Printf("failed loading cassette %s': %s\n", cassetteName, err.Error())
			return
		}
		vcrConfig.cassette = k7
	}
}

// VCRSettings holds a set of options for the VCR.
type VCRSettings struct {
	client   *http.Client
	cassette *cassette

	// trackRecordingMutator mutatorOfSomeSortThatTakesATrack // only the exported fields of the track will be mutable, the others will be invisible
	// trackReplayingMutator mutatorOfSomeSortThatTakesATrack // only the exported fields of the track will be mutable, the others will be invisible
}
