package govcr

import "net/http"

// NewVCR creates a new VCR.
func NewVCR(settings ...Setting) *ControlPanel {
	var vcrSettings VCRSettings

	for _, option := range settings {
		option(&vcrSettings)
	}

	// use a default client if none provided
	if vcrSettings.client == nil {
		vcrSettings.client = http.DefaultClient
	}

	// use a default vcrTransport if none provided
	if vcrSettings.client.Transport == nil {
		vcrSettings.client.Transport = http.DefaultTransport
	}

	// create VCR's HTTP client
	vcrClient := &http.Client{
		Transport: &vcrTransport{
			pcb:       &pcb{},
			cassette:  vcrSettings.cassette,
			transport: vcrSettings.client.Transport,
		},
	}

	// copy the attributes of the original http.Client
	vcrClient.CheckRedirect = vcrSettings.client.CheckRedirect
	vcrClient.Jar = vcrSettings.client.Jar
	vcrClient.Timeout = vcrSettings.client.Timeout

	// return
	return &ControlPanel{
		client: vcrClient,
	}
}
