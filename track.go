package govcr

import (
	"errors"
	"fmt"
	"net"
)

// track is a recording (HTTP request + response) in a cassette.
type Track struct {
	Request  Request
	Response Response
	ErrType  string
	ErrMsg   string

	// replayed indicates whether the track has already been processed in the cassette playback.
	replayed bool
}

func NewTrack(req *Request, resp *Response, reqErr error) *Track {
	// record error type, if error
	var reqErrType, reqErrMsg string
	if reqErr != nil {
		reqErrType = fmt.Sprintf("%T", reqErr)
		reqErrMsg = reqErr.Error()
	}

	var reqValue Request
	if req != nil {
		reqValue = *req
	}

	var respValue Response
	if resp != nil {
		respValue = *resp
	}

	track := &Track{
		Request:  reqValue,
		Response: respValue,
		ErrType:  reqErrType,
		ErrMsg:   reqErrMsg,
	}

	return track
}

func (t *Track) response() (*Response, error) {
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
