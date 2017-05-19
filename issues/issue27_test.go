package issues_test

import (
	"testing"

	"github.com/parnurzeal/gorequest"
	"github.com/seborama/govcr"
)

func TestIssue27(t *testing.T) {
	gorequest.DisableTransportSwap = true
	vcr := govcr.NewVCR("cassette", &govcr.VCRConfig{
		ExcludeHeaderFunc: func(key string) bool {
			return true
		},
		Logging: true})

	request := gorequest.New()
	request.Client = vcr.Client
	resp, body, errs := request.Get("https://example.com").End()
	if errs != nil {
		t.Fatalf("Get returned an error: %+v\n", errs)
	}
	t.Logf("body: %+v\n", body)
	t.Logf("resp: %+v\n", resp)
}
