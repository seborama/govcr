package track

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"io/ioutil"
	"log"
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
	Method           string
	URL              *url.URL
	Proto            string
	ProtoMajor       int
	ProtoMinor       int
	Header           http.Header
	Body             []byte
	ContentLength    int64
	TransferEncoding []string
	Close            bool
	Host             string
	Form             url.Values
	PostForm         url.Values
	MultipartForm    *multipart.Form // attachments that get offset to temp files may not be supported (untested)
	Trailer          http.Header
	RemoteAddr       string
	RequestURI       string
	// TODO: Response ?
}

// ToRequest transcodes an HTTP Request to a track Request.
func ToRequest(httpRequest *http.Request) *Request {
	if httpRequest == nil {
		return nil
	}

	// deal with body first because Trailers are sent after Body.Read returns io.EOF and Body.Close() was called.
	bodyClone := cloneHTTPRequestBody(httpRequest)
	headerClone := cloneHeader(httpRequest.Header)
	trailerClone := cloneHeader(httpRequest.Trailer)

	return &Request{
		Method:        httpRequest.Method,
		URL:           cloneURL(httpRequest.URL),
		Proto:         httpRequest.Proto,
		ProtoMajor:    httpRequest.ProtoMajor,
		ProtoMinor:    httpRequest.ProtoMinor,
		Header:        headerClone,
		Body:          bodyClone,
		ContentLength: httpRequest.ContentLength,
		// TODO: TransferEncoding: []string{},
		Close: httpRequest.Close,
		Host:  httpRequest.Host,
		// TODO: Form:          map[string][]string{},
		// TODO: PostForm:      map[string][]string{},
		// TODO: MultipartForm: &multipart.Form{},
		Trailer:    trailerClone,
		RemoteAddr: httpRequest.RemoteAddr,
		RequestURI: httpRequest.RequestURI,
	}
}

// Response is a track HTTP Response.
type Response struct {
	Status     string
	StatusCode int
	Proto      string
	ProtoMajor int
	ProtoMinor int

	Header           http.Header
	Body             []byte
	ContentLength    int64
	TransferEncoding []string
	Trailer          http.Header
	TLS              *tls.ConnectionState
}

// ToResponse transcodes an HTTP Response to a track Response.
func ToResponse(httpResponse *http.Response) *Response {
	if httpResponse == nil {
		return nil
	}

	// deal with body first because Trailers are sent after Body.Read returns io.EOF and Body.Close() was called.
	bodyClone := cloneHTTPResponseBody(httpResponse)
	headerClone := cloneHeader(httpResponse.Header)
	trailerClone := cloneHeader(httpResponse.Trailer)
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
		Trailer:          trailerClone,
		TLS:              tlsClone,
	}
}

func cloneTLS(tlsCS *tls.ConnectionState) *tls.ConnectionState {
	if tlsCS == nil {
		return nil
	}

	var signedCertificateTimestampsClone [][]byte //nolint:prealloc
	for _, data := range tlsCS.SignedCertificateTimestamps {
		signedCertificateTimestampsClone = append(signedCertificateTimestampsClone, []byte(string(data)))
	}

	var peerCertificatesClone []*x509.Certificate
	if err := copier.Copy(&peerCertificatesClone, tlsCS.PeerCertificates); err != nil {
		log.Println("failed to deep copy tlsCS.PeerCertificates: " + err.Error())

		peerCertificatesClone = tlsCS.PeerCertificates
	}

	var verifiedChainsClone [][]*x509.Certificate //nolint:prealloc
	for _, certSlice := range tlsCS.VerifiedChains {
		var certSliceClone []*x509.Certificate

		if err := copier.Copy(&certSliceClone, certSlice); err != nil {
			log.Println("failed to deep copy tlsCS.VerifiedChains: " + err.Error())

			verifiedChainsClone = tlsCS.VerifiedChains

			break
		}

		verifiedChainsClone = append(verifiedChainsClone, certSliceClone)
	}

	return &tls.ConnectionState{
		Version:                     tlsCS.Version,
		HandshakeComplete:           tlsCS.HandshakeComplete,
		DidResume:                   tlsCS.DidResume,
		CipherSuite:                 tlsCS.CipherSuite,
		NegotiatedProtocol:          tlsCS.NegotiatedProtocol,
		NegotiatedProtocolIsMutual:  tlsCS.NegotiatedProtocolIsMutual, //nolint: staticcheck
		ServerName:                  tlsCS.ServerName,
		PeerCertificates:            peerCertificatesClone,
		VerifiedChains:              verifiedChainsClone,
		SignedCertificateTimestamps: signedCertificateTimestampsClone,
		OCSPResponse:                []byte(string(tlsCS.OCSPResponse)),
		TLSUnique:                   []byte(string(tlsCS.TLSUnique)), //nolint: staticcheck
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

		httpBodyClone, err = ioutil.ReadAll(httpRequest.Body)
		if err != nil {
			log.Println("cloneHTTPRequestBody - httpBodyClone:", err)
		}

		err = httpRequest.Body.Close()
		if err != nil {
			log.Println("cloneHTTPRequestBody - httpRequest.Body.Close:", err)
		}

		httpRequest.Body = ioutil.NopCloser(bytes.NewBuffer(httpBodyClone))
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

		httpBodyClone, err = ioutil.ReadAll(httpResponse.Body)
		if err != nil {
			log.Println("cloneHTTPResponseBody - httpBodyClone:", err)
		}

		err = httpResponse.Body.Close()
		if err != nil {
			log.Println("cloneHTTPResponseBody - httpResponse.Body.Close:", err)
		}

		httpResponse.Body = ioutil.NopCloser(bytes.NewBuffer(httpBodyClone))
	}

	return httpBodyClone
}

func cloneHeader(headers http.Header) http.Header {
	if headers == nil {
		return nil
	}

	headersClone := make(http.Header)
	for key, value := range headers {
		headersClone[key] = make([]string, len(value))
		copy(headersClone[key], value)
	}
	return headersClone
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
		Scheme:     aURL.Scheme,
		Opaque:     aURL.Opaque,
		User:       user,
		Host:       aURL.Host,
		Path:       aURL.Path,
		RawPath:    aURL.RawPath,
		ForceQuery: aURL.ForceQuery,
		RawQuery:   aURL.RawQuery,
		Fragment:   aURL.Fragment,
		// TODO: RawFragment?
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
	httpRequestClone.Cancel = nil //nolint:staticcheck

	// deal with the URL
	if httpRequest.URL != nil {
		httpRequestClone.URL = cloneURL(httpRequest.URL)
	}

	// deal with body first because Trailers are sent after Body.Read returns io.EOF and Body.Close() was called.
	httpRequestClone.Body = ioutil.NopCloser(bytes.NewBuffer(cloneHTTPRequestBody(httpRequest)))
	httpRequestClone.Header = cloneHeader(httpRequest.Header)
	httpRequestClone.Trailer = cloneHeader(httpRequest.Trailer)
	httpRequestClone.TransferEncoding = cloneStringSlice(httpRequest.TransferEncoding)
	httpRequestClone.Form = cloneURLValues(httpRequest.Form)
	httpRequestClone.PostForm = cloneURLValues(httpRequest.PostForm)

	// TODO:
	// MultipartForm
	httpRequestClone.TLS = cloneTLS(httpRequest.TLS)

	var responseClone *http.Response
	if httpRequest.Response != nil {
		if err := copier.Copy(&responseClone, httpRequest.Response); err != nil {
			log.Println("cannot deep copy httpRequest.Response: " + err.Error())

			responseClone = httpRequest.Response // TODO: if ever creating a cloneHTTPResponse() function, use it!
		}
	}

	httpRequestClone.Response = responseClone

	return &httpRequestClone
}
