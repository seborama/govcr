package track

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"net/http"

	"github.com/pkg/errors"

	trkerr "github.com/seborama/govcr/v13/cassette/track/errors"
)

// Track is a recording (HTTP Request + Response) in a cassette.
type Track struct {
	Request  Request
	Response *Response
	ErrType  *string
	ErrMsg   *string
	UUID     string // future enhancement to identify tracks in logs, etc

	// replayed indicates whether the track has already been processed in the cassette playback.
	replayed bool
}

// NewTrack creates a new Track.
func NewTrack(req *Request, resp *Response, reqErr error) *Track {
	// record error type, if error
	var reqErrType, reqErrMsg *string
	if reqErr != nil {
		reqErrType = strPtr(fmt.Sprintf("%T", reqErr))
		reqErrMsg = strPtr(reqErr.Error())
	}

	var reqValue Request
	if req != nil {
		reqValue = *req
	}

	track := &Track{
		Request:  reqValue,
		Response: resp,
		ErrType:  reqErrType,
		ErrMsg:   reqErrMsg,
		replayed: false,
	}

	return track
}

// IsReplayed returns true if the Track has already been replayed, otherwise
// it returns false.
func (trk *Track) IsReplayed() bool {
	return trk.replayed
}

// SetReplayed sets the replays status of the track.
func (trk *Track) SetReplayed(replayed bool) {
	trk.replayed = replayed
}

// ToErr converts the track Err to an http.Response.
func (trk *Track) ToErr() error {
	if trk.ErrType == nil {
		return nil
	}

	errType := *trk.ErrType

	errMsg := ""
	if trk.ErrMsg != nil {
		errMsg = *trk.ErrMsg
	}

	if errType == "*net.OpError" {
		return &net.OpError{
			Op:     "govcr",
			Net:    "govcr",
			Source: nil,
			Addr:   nil,
			Err:    errors.WithStack(trkerr.NewErrTransportFailure(errType, errMsg)),
		}
	}

	return errors.WithStack(trkerr.NewErrTransportFailure(errType, errMsg))
}

// toHTTPRequest converts the track Request to an http.Request.
// NOTE:
// govcr only saves enough info of the http.Request to permit matching.
// Not all fields of http.Request are populated.
func (trk *Track) toHTTPRequest() *http.Request {
	bodyReadCloser := io.NopCloser(bytes.NewReader(trk.Request.Body))

	httpRequest := http.Request{
		Method:           trk.Request.Method,
		URL:              trk.Request.URL,
		Proto:            trk.Request.Proto,
		ProtoMajor:       trk.Request.ProtoMajor,
		ProtoMinor:       trk.Request.ProtoMinor,
		Header:           trk.Request.Header,
		Body:             bodyReadCloser,
		ContentLength:    trk.Request.ContentLength,
		TransferEncoding: trk.Request.TransferEncoding,
		Close:            trk.Request.Close,
		Host:             trk.Request.Host,
		Form:             trk.Request.Form,
		PostForm:         trk.Request.PostForm,
		MultipartForm:    trk.Request.MultipartForm,
		Trailer:          trk.Request.Trailer,
		RemoteAddr:       trk.Request.RemoteAddr,
		RequestURI:       trk.Request.RequestURI,
	}

	return &httpRequest
}

// ToHTTPResponse converts the track Response to an http.Response.
// nolint: gocritic
func (trk Track) ToHTTPResponse() *http.Response {
	if trk.Response == nil {
		return nil
	}

	httpResponse := http.Response{}

	bodyReadCloser := io.NopCloser(bytes.NewReader(trk.Response.Body))

	httpResponse.Status = trk.Response.Status
	httpResponse.StatusCode = trk.Response.StatusCode
	httpResponse.Proto = trk.Response.Proto
	httpResponse.ProtoMajor = trk.Response.ProtoMajor
	httpResponse.ProtoMinor = trk.Response.ProtoMinor

	httpResponse.Header = trk.Response.Header
	httpResponse.Body = bodyReadCloser
	httpResponse.ContentLength = trk.Response.ContentLength
	httpResponse.TransferEncoding = trk.Response.TransferEncoding
	httpResponse.Close = trk.Response.Close
	httpResponse.Uncompressed = trk.Response.Uncompressed
	httpResponse.Trailer = trk.Response.Trailer
	httpResponse.TLS = trk.Response.TLS

	httpResponse.Request = trk.toHTTPRequest()
	httpResponse.Request.Body = nil // See notes on http.response.request - Body is nil because it has already been consumed

	return &httpResponse
}

func strPtr(s string) *string { return &s }
