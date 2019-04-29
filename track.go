package govcr

import (
	"errors"
	"fmt"
	"net"
)

// track is a recording (HTTP request + response) in a cassette.
type track struct {
	Request  request
	Response response
	ErrType  string
	ErrMsg   string

	// replayed indicates whether the track has already been processed in the cassette playback.
	replayed bool
}

func newTrack(req *request, resp *response, reqErr error) (*track, error) {
	// record error type, if error
	var reqErrType, reqErrMsg string
	if reqErr != nil {
		reqErrType = fmt.Sprintf("%T", reqErr)
		reqErrMsg = reqErr.Error()
	}

	track := &track{
		Request:  *req,
		Response: *resp,
		ErrType:  reqErrType,
		ErrMsg:   reqErrMsg,
	}

	return track, nil
}

func (t *track) response() (*response, error) {
	var err error

	// create error object
	switch t.ErrType {
	case "*net.OpError":
		err = &net.OpError{
			Op:     "govcr",
			Net:    "govcr",
			Source: nil,
			Addr:   nil,
			Err:    errors.New(t.ErrType + ": " + t.ErrMsg),
		}
	case "":
		err = nil

	default:
		err = errors.New(t.ErrType + ": " + t.ErrMsg)
	}

	if err != nil {
		// No need to parse the response.
		// By convention, when an HTTP error occurred, the response should be nil
		// (or Go's http package will show a warning message at runtime).
		return nil, err
	}

	return &t.Response, nil
}
