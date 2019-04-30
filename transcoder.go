package govcr

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"

	"github.com/jinzhu/copier"
)

// request is a recorded HTTP request.
type request struct {
	Method  string
	URL     *url.URL
	Header  http.Header
	Body    []byte
	Trailer http.Header
}

func fromHTTPRequest(httpRequest *http.Request) *request {
	headerClone := cloneHeader(httpRequest.Header)
	trailerClone := cloneHeader(httpRequest.Trailer)
	bodyClone := cloneHTTPRequestBody(httpRequest)

	return &request{
		Method:  httpRequest.Method,
		URL:     cloneURL(httpRequest.URL),
		Header:  headerClone,
		Body:    bodyClone,
		Trailer: trailerClone,
	}
}

// response is a recorded HTTP response.
type response struct {
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

func fromHTTPResponse(httpResponse *http.Response) *response {
	headerClone := cloneHeader(httpResponse.Header)
	trailerClone := cloneHeader(httpResponse.Trailer)
	bodyClone := cloneHTTPResponseBody(httpResponse)
	tsfEncodingClone := cloneStringSlice(httpResponse.TransferEncoding)

	tlsClone := cloneTLS(httpResponse.TLS)

	return &response{
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
	var signedCertificateTimestampsClone [][]byte
	for _, data := range tlsCS.SignedCertificateTimestamps {
		signedCertificateTimestampsClone = append(signedCertificateTimestampsClone, []byte(string(data)))
	}

	var peerCertificatesClone []*x509.Certificate
	if err := copier.Copy(&peerCertificatesClone, tlsCS.PeerCertificates); err != nil {
		log.Println("cannot deep copy tlsCS.PeerCertificates: " + err.Error())
		peerCertificatesClone = tlsCS.PeerCertificates
	}

	var verifiedChainsClone [][]*x509.Certificate
	if err := copier.Copy(&verifiedChainsClone, tlsCS.VerifiedChains); err != nil {
		log.Println("cannot deep copy tlsCS.VerifiedChains: " + err.Error())
		verifiedChainsClone = tlsCS.VerifiedChains
	}

	return &tls.ConnectionState{
		Version:                     tlsCS.Version,
		HandshakeComplete:           tlsCS.HandshakeComplete,
		DidResume:                   tlsCS.DidResume,
		CipherSuite:                 tlsCS.CipherSuite,
		NegotiatedProtocol:          tlsCS.NegotiatedProtocol,
		NegotiatedProtocolIsMutual:  tlsCS.NegotiatedProtocolIsMutual,
		ServerName:                  tlsCS.ServerName,
		PeerCertificates:            peerCertificatesClone,
		VerifiedChains:              verifiedChainsClone,
		SignedCertificateTimestamps: signedCertificateTimestampsClone,
		OCSPResponse:                []byte(string(tlsCS.OCSPResponse)),
		TLSUnique:                   []byte(string(tlsCS.TLSUnique)),
	}
}

func cloneStringSlice(stringSlice []string) []string {
	stringSliceClone := make([]string, len(stringSlice))
	copy(stringSliceClone, stringSlice)
	return stringSliceClone
}

// toHTTPResponse convert a response to an HTTP.Response.
// Note that this function sets HTTP.Response.Request to nil.
func toHTTPResponse(response *response) *http.Response {
	httpResponse := http.Response{}

	// create a ReadCloser to supply to httpResponse
	bodyReadCloser := ioutil.NopCloser(bytes.NewReader(response.Body))

	// re-create the response object from track record
	respTLS := response.TLS

	httpResponse.Status = response.Status
	httpResponse.StatusCode = response.StatusCode
	httpResponse.Proto = response.Proto
	httpResponse.ProtoMajor = response.ProtoMajor
	httpResponse.ProtoMinor = response.ProtoMinor

	httpResponse.Header = response.Header
	httpResponse.Body = bodyReadCloser
	httpResponse.ContentLength = response.ContentLength
	httpResponse.TransferEncoding = response.TransferEncoding
	httpResponse.Trailer = response.Trailer
	httpResponse.TLS = respTLS

	return &httpResponse
}

func cloneHTTPRequestBody(httpRequest *http.Request) []byte {
	var httpBodyClone []byte
	if httpRequest.Body != nil {
		httpBodyClone, _ = ioutil.ReadAll(httpRequest.Body)
		_ = httpRequest.Body.Close()
		httpRequest.Body = ioutil.NopCloser(bytes.NewBuffer(httpBodyClone))
	}

	return httpBodyClone
}

func cloneHTTPResponseBody(httpResponse *http.Response) []byte {
	var httpBodyClone []byte
	if httpResponse.Body != nil {
		httpBodyClone, _ = ioutil.ReadAll(httpResponse.Body)
		_ = httpResponse.Body.Close()
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
	}
}

func cloneHTTPRequest(httpRequest *http.Request) *http.Request {
	if httpRequest == nil {
		return nil
	}

	// get a shallow copy
	httpRequestClone := *httpRequest

	// remove the channel reference
	httpRequestClone.Cancel = nil

	// deal with the URL
	if httpRequest.URL != nil {
		httpRequestClone.URL = cloneURL(httpRequest.URL)
	}
	httpRequestClone.Header = cloneHeader(httpRequest.Header)
	httpRequestClone.Body = ioutil.NopCloser(bytes.NewBuffer(cloneHTTPRequestBody(httpRequest)))
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
