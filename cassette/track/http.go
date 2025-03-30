package track

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"

	"github.com/jinzhu/copier"
)

// Request is a track HTTP Request.
// Several of these fields are present for use with mutators rather than
// with a RequestMatcher (albeit perfectly possible).
// These fields also help when converting Response to http.Response to
// populate http.Response.Request.
type Request struct {
	Method           string          `json:"Method"`
	URL              *url.URL        `json:"URL"`
	Proto            string          `json:"Proto"`
	ProtoMajor       int             `json:"ProtoMajor"`
	ProtoMinor       int             `json:"ProtoMinor"`
	Header           http.Header     `json:"Header"`
	Body             []byte          `json:"Body"`
	ContentLength    int64           `json:"ContentLength"`
	TransferEncoding []string        `json:"TransferEncoding"`
	Close            bool            `json:"Close"`
	Host             string          `json:"Host"`
	Form             url.Values      `json:"Form"`
	PostForm         url.Values      `json:"PostForm"`
	MultipartForm    *multipart.Form `json:"MultipartForm"` // attachments that get offset to temp files may not be supported (untested)
	Trailer          http.Header     `json:"Trailer"`
	RemoteAddr       string          `json:"RemoteAddr"`
	RequestURI       string          `json:"RequestURI"`
	// TODO: Response ?
}

// Clone returns a copy of r or nil if r is nil.
// Note: this is inaccurate: MultipartForm cannot be truly cloned.
func (r *Request) Clone() *Request {
	if r == nil {
		return nil
	}

	body := make([]byte, len(r.Body))
	copy(body, r.Body)

	newR := &Request{
		Method:           r.Method,
		URL:              cloneURL(r.URL),
		Proto:            r.Proto,
		ProtoMajor:       r.ProtoMajor,
		ProtoMinor:       r.ProtoMinor,
		Header:           r.Header.Clone(),
		Body:             body,
		ContentLength:    r.ContentLength,
		TransferEncoding: cloneStringSlice(r.TransferEncoding),
		Close:            r.Close,
		Host:             r.Host,
		Form:             cloneMapOfSlices(r.Form),
		PostForm:         cloneMapOfSlices(r.PostForm),
		MultipartForm:    cloneMultipartForm(r.MultipartForm),
		Trailer:          cloneMapOfSlices(r.Trailer),
		RemoteAddr:       r.RemoteAddr,
		RequestURI:       r.RequestURI,
	}

	return newR
}

func cloneMultipartForm(src *multipart.Form) *multipart.Form {
	if src == nil {
		return nil
	}

	dst := &multipart.Form{
		Value: cloneMapOfSlices(src.Value),
		File:  cloneMultipartFormFile(src.File),
	}

	return dst
}

// Note: this is inaccurate: FileHeader cannot be truly cloned.
func cloneMultipartFormFile(src map[string][]*multipart.FileHeader) map[string][]*multipart.FileHeader {
	if src == nil {
		return nil
	}

	dst := map[string][]*multipart.FileHeader{}

	for k, v := range src {
		dst[k] = cloneSliceOfMultipartFileHeader(v)
	}

	return dst
}

// Note: this is inaccurate: FileHeader cannot be truly cloned.
func cloneSliceOfMultipartFileHeader(src []*multipart.FileHeader) []*multipart.FileHeader {
	if src == nil {
		return src
	}

	dst := make([]*multipart.FileHeader, len(src))

	for k, v := range src {
		dst[k] = &multipart.FileHeader{
			Filename: v.Filename,
			Header:   cloneMapOfSlices(v.Header),
			Size:     v.Size,
		}
	}

	return dst
}

func cloneMapOfSlices(src map[string][]string) map[string][]string {
	if src == nil {
		return nil
	}

	dst := map[string][]string{}

	for k, v := range src {
		vCopy := make([]string, len(v))
		copy(vCopy, v)

		dst[k] = vCopy
	}

	return dst
}

// ToRequest transcodes an HTTP Request to a track Request.
func ToRequest(httpRequest *http.Request) *Request {
	if httpRequest == nil {
		return nil
	}

	// deal with body first because Trailers are sent after Body.Read returns io.EOF and Body.Close() was called.
	bodyClone := cloneHTTPRequestBody(httpRequest)
	headerClone := httpRequest.Header.Clone()
	trailerClone := httpRequest.Trailer.Clone()
	tsfEncodingClone := cloneStringSlice(httpRequest.TransferEncoding)

	return &Request{
		Method:           httpRequest.Method,
		URL:              cloneURL(httpRequest.URL),
		Proto:            httpRequest.Proto,
		ProtoMajor:       httpRequest.ProtoMajor,
		ProtoMinor:       httpRequest.ProtoMinor,
		Header:           headerClone,
		Body:             bodyClone,
		ContentLength:    httpRequest.ContentLength,
		TransferEncoding: tsfEncodingClone,
		Close:            httpRequest.Close,
		Host:             httpRequest.Host,
		Form:             cloneMapOfSlices(httpRequest.Form),
		PostForm:         cloneMapOfSlices(httpRequest.PostForm),
		MultipartForm:    cloneMultipartForm(httpRequest.MultipartForm),
		Trailer:          trailerClone,
		RemoteAddr:       httpRequest.RemoteAddr,
		RequestURI:       httpRequest.RequestURI,
	}
}

// Response is a track HTTP Response.
type Response struct {
	Status     string `json:"Status"`
	StatusCode int    `json:"StatusCode"`
	Proto      string `json:"Proto"`
	ProtoMajor int    `json:"ProtoMajor"`
	ProtoMinor int    `json:"ProtoMinor"`

	Header           http.Header          `json:"Header"`
	Body             []byte               `json:"Body"`
	ContentLength    int64                `json:"ContentLength"`
	TransferEncoding []string             `json:"TransferEncoding"`
	Close            bool                 `json:"Close"`
	Uncompressed     bool                 `json:"Uncompressed"`
	Trailer          http.Header          `json:"Trailer"`
	TLS              *tls.ConnectionState `json:"TLS"`

	// govcr initially sets Request to nil when recording a track to the cassette.
	// It is only possible to force a value to cassette through a track record mutator.
	// At _replaying_ _time_ _only_ it will be populated with the "current" HTTP request.
	// This is useful in scenarios where the request contains a dynamic piece of information
	// such as e.g. a transaction ID, a customer number, etc.
	// This is solely for informational purpose at replaying time.
	// Mutating it at replay time typically achieves nothing very useful.
	Request *Request `json:"Request"`
}

// ToResponse transcodes an HTTP Response to a track Response.
func ToResponse(httpResponse *http.Response) *Response {
	if httpResponse == nil {
		return nil
	}

	// deal with body first because Trailers are sent after Body.Read returns io.EOF and Body.Close() was called.
	bodyClone := cloneHTTPResponseBody(httpResponse)
	headerClone := httpResponse.Header.Clone()
	trailerClone := httpResponse.Trailer.Clone()
	tsfEncodingClone := cloneStringSlice(httpResponse.TransferEncoding)

	tlsClone := cloneTLS(httpResponse.TLS)

	return &Response{
		Status:           httpResponse.Status,
		StatusCode:       httpResponse.StatusCode,
		Proto:            httpResponse.Proto,
		ProtoMajor:       httpResponse.ProtoMajor,
		ProtoMinor:       httpResponse.ProtoMinor,
		Header:           headerClone,
		Body:             bodyClone,
		ContentLength:    httpResponse.ContentLength,
		TransferEncoding: tsfEncodingClone,
		Close:            httpResponse.Close,
		Uncompressed:     httpResponse.Uncompressed,
		Trailer:          trailerClone,
		TLS:              tlsClone,
	}
}

func cloneTLS(tlsCS *tls.ConnectionState) *tls.ConnectionState {
	if tlsCS == nil {
		return nil
	}

	var signedCertificateTimestampsClone [][]byte
	for _, data := range tlsCS.SignedCertificateTimestamps {
		signedCertificateTimestampsClone = append(signedCertificateTimestampsClone, []byte(string(data)))
	}

	var peerCertificatesClone []*x509.Certificate
	if err := copier.Copy(&peerCertificatesClone, tlsCS.PeerCertificates); err != nil {
		slog.Info("failed to deep copy tlsCS.PeerCertificates", slog.Any("error", err))

		peerCertificatesClone = tlsCS.PeerCertificates
	}

	removePublicKey(peerCertificatesClone)

	var verifiedChainsClone [][]*x509.Certificate
	for _, certSlice := range tlsCS.VerifiedChains {
		var certSliceClone []*x509.Certificate

		if err := copier.Copy(&certSliceClone, certSlice); err != nil {
			slog.Error("failed to deep copy tlsCS.VerifiedChains", slog.Any("error", err.Error()))

			verifiedChainsClone = tlsCS.VerifiedChains

			break
		}

		removePublicKey(certSliceClone)

		verifiedChainsClone = append(verifiedChainsClone, certSliceClone)
	}

	return &tls.ConnectionState{
		Version:                     tlsCS.Version,
		HandshakeComplete:           tlsCS.HandshakeComplete,
		DidResume:                   tlsCS.DidResume,
		CipherSuite:                 tlsCS.CipherSuite,
		NegotiatedProtocol:          tlsCS.NegotiatedProtocol,
		NegotiatedProtocolIsMutual:  tlsCS.NegotiatedProtocolIsMutual, //nolint:staticcheck // deprecated but still present
		ServerName:                  tlsCS.ServerName,
		PeerCertificates:            peerCertificatesClone,
		VerifiedChains:              verifiedChainsClone,
		SignedCertificateTimestamps: signedCertificateTimestampsClone,
		OCSPResponse:                []byte(string(tlsCS.OCSPResponse)),
		TLSUnique:                   []byte(string(tlsCS.TLSUnique)),
	}
}

func removePublicKey(certs []*x509.Certificate) {
	for i := range certs {
		// destroy PublicKey as it's untyped and breaks with the json package.
		certs[i].PublicKey = nil
	}
}

func cloneStringSlice(stringSlice []string) []string {
	stringSliceClone := make([]string, len(stringSlice))
	copy(stringSliceClone, stringSlice)

	return stringSliceClone
}

func cloneHTTPRequestBody(httpRequest *http.Request) []byte {
	if httpRequest == nil {
		return nil
	}

	var httpBodyClone []byte
	if httpRequest.Body != nil {
		var err error

		httpBodyClone, err = io.ReadAll(httpRequest.Body)
		if err != nil {
			slog.Error("cloneHTTPRequestBody - httpBodyClone", slog.String("error", err.Error()))
		}

		err = httpRequest.Body.Close()
		if err != nil {
			slog.Error("cloneHTTPRequestBody - httpRequest.Body.Close", slog.String("error", err.Error()))
		}

		httpRequest.Body = io.NopCloser(bytes.NewBuffer(httpBodyClone))
	}

	return httpBodyClone
}

func cloneHTTPResponseBody(httpResponse *http.Response) []byte {
	if httpResponse == nil {
		return nil
	}

	var httpBodyClone []byte
	if httpResponse.Body != nil {
		var err error

		httpBodyClone, err = io.ReadAll(httpResponse.Body)
		if err != nil {
			slog.Error("cloneHTTPResponseBody - httpBodyClone", slog.String("error", err.Error()))
		}

		err = httpResponse.Body.Close()
		if err != nil {
			slog.Error("cloneHTTPResponseBody - httpResponse.Body.Close", slog.String("error", err.Error()))
		}

		httpResponse.Body = io.NopCloser(bytes.NewBuffer(httpBodyClone))
	}

	return httpBodyClone
}

func cloneURLValues(urlValues url.Values) url.Values {
	if urlValues == nil {
		return nil
	}

	urlValuesClone := make(url.Values)
	for key, value := range urlValues {
		urlValuesClone[key] = make([]string, len(value))
		copy(urlValuesClone[key], value)
	}

	return urlValuesClone
}

func cloneURL(aURL *url.URL) *url.URL {
	if aURL == nil {
		return nil
	}

	var user *url.Userinfo

	if aURL.User != nil {
		userPassword := strings.SplitN(aURL.User.String(), ":", 2)
		if len(userPassword) == 1 {
			user = url.User(userPassword[0])
		} else {
			user = url.UserPassword(userPassword[0], userPassword[1])
		}
	}

	return &url.URL{
		Scheme:      aURL.Scheme,
		Opaque:      aURL.Opaque,
		User:        user,
		Host:        aURL.Host,
		Path:        aURL.Path,
		RawPath:     aURL.RawPath,
		ForceQuery:  aURL.ForceQuery,
		RawQuery:    aURL.RawQuery,
		Fragment:    aURL.Fragment,
		RawFragment: aURL.RawFragment,
	}
}

// CloneHTTPRequest clones an http.Request.
func CloneHTTPRequest(httpRequest *http.Request) *http.Request {
	if httpRequest == nil {
		return nil
	}

	// get a shallow copy
	httpRequestClone := *httpRequest

	// remove the channel reference
	httpRequestClone.Cancel = nil //nolint:staticcheck // deprecated but still present

	// deal with the URL
	if httpRequest.URL != nil {
		httpRequestClone.URL = cloneURL(httpRequest.URL)
	}

	// deal with body first because Trailers are sent after Body.Read returns io.EOF and Body.Close() was called.
	httpRequestClone.Body = io.NopCloser(bytes.NewBuffer(cloneHTTPRequestBody(httpRequest)))
	httpRequestClone.Header = httpRequest.Header.Clone()
	httpRequestClone.Trailer = httpRequest.Trailer.Clone()
	httpRequestClone.TransferEncoding = cloneStringSlice(httpRequest.TransferEncoding)
	httpRequestClone.Form = cloneURLValues(httpRequest.Form)
	httpRequestClone.PostForm = cloneURLValues(httpRequest.PostForm)
	httpRequestClone.MultipartForm = cloneMultipartForm(httpRequest.MultipartForm)
	httpRequestClone.TLS = cloneTLS(httpRequest.TLS)

	var responseClone *http.Response
	if httpRequest.Response != nil {
		if err := copier.Copy(&responseClone, httpRequest.Response); err != nil {
			slog.Info("cannot deep copy httpRequest.Response", slog.String("error", err.Error()))

			responseClone = httpRequest.Response // TODO: if ever creating a cloneHTTPResponse() function, use it!
		}
	}

	httpRequestClone.Response = responseClone

	return &httpRequestClone
}
