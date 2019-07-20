package cassette

import (
	"errors"
	"fmt"
	"net"
)

// Track is a recording (HTTP Request + Response) in a cassette.
type Track struct {
	Request  Request
	Response Response
	ErrType  string
	ErrMsg   string

	// replayed indicates whether the track has already been processed in the cassette playback.
	replayed bool
}

// NewTrack creates a new Track.
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

// GetRequest returns the HTTP Request object of this track.
func (t *Track) GetRequest() *Request {
	return &t.Request
}

// GetResponse returns the HTTP Response object of this track.
func (t *Track) GetResponse() (*Response, error) {
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
		// No need to parse the Response.
		// By convention, when an HTTP error occurred, the Response should be nil
		// (or Go's http package will show a warning message at runtime).
		return nil, err
	}

	return &t.Response, nil
}

func (t *Track) IsReplayed() bool {
	return t.replayed
}

func (t *Track) Replayed(replayed bool) {
	t.replayed = replayed
}
