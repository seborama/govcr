package main

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMain_decryptCommand(t *testing.T) {
	got, err := decryptCassette("./test-fixtures/TestExample4.cassette.json", "./test-fixtures/TestExample4.unsafe.key")
	require.NoError(t, err)

	expected := `{
  "Tracks": [
    {
      "Request": {
        "Method": "GET",
        "URL": {
          "Scheme": "http",
          "Opaque": "",
          "User": null,
          "Host": "example.com",
          "Path": "/foo",
          "RawPath": "",
          "ForceQuery": false,
          "RawQuery": "",
          "Fragment": "",
          "RawFragment": ""
        },
        "Proto": "HTTP/1.1",
        "ProtoMajor": 1,
        "ProtoMinor": 1,
        "Header": {},
        "Body": "",
        "ContentLength": 0,
        "TransferEncoding": [],
        "Close": false,
        "Host": "example.com",
        "Form": null,
        "PostForm": null,
        "MultipartForm": null,
        "Trailer": null,
        "RemoteAddr": "",
        "RequestURI": ""
      },
      "Response": {
        "Status": "404 Not Found",
        "StatusCode": 404,
        "Proto": "HTTP/1.1",
        "ProtoMajor": 1,
        "ProtoMinor": 1,
        "Header": {
          "Accept-Ranges": [
            "bytes"
          ],
          "Age": [
            "527190"
          ],
          "Cache-Control": [
            "max-age=604800"
          ],
          "Content-Type": [
            "text/html; charset=UTF-8"
          ],
          "Date": [
            "Sun, 14 Aug 2022 18:12:40 GMT"
          ],
          "Expires": [
            "Sun, 21 Aug 2022 18:12:40 GMT"
          ],
          "Last-Modified": [
            "Mon, 08 Aug 2022 15:46:10 GMT"
          ],
          "Server": [
            "ECS (bsa/EB1E)"
          ],
          "Vary": [
            "Accept-Encoding"
          ],
          "X-Cache": [
            "404-HIT"
          ]
        },
        "Body": "PCFkb2N0eXBlIGh0bWw+CjxodG1sPgo8aGVhZD4KICAgIDx0aXRsZT5FeGFtcGxlIERvbWFpbjwvdGl0bGU+CgogICAgPG1ldGEgY2hhcnNldD0idXRmLTgiIC8+CiAgICA8bWV0YSBodHRwLWVxdWl2PSJDb250ZW50LXR5cGUiIGNvbnRlbnQ9InRleHQvaHRtbDsgY2hhcnNldD11dGYtOCIgLz4KICAgIDxtZXRhIG5hbWU9InZpZXdwb3J0IiBjb250ZW50PSJ3aWR0aD1kZXZpY2Utd2lkdGgsIGluaXRpYWwtc2NhbGU9MSIgLz4KICAgIDxzdHlsZSB0eXBlPSJ0ZXh0L2NzcyI+CiAgICBib2R5IHsKICAgICAgICBiYWNrZ3JvdW5kLWNvbG9yOiAjZjBmMGYyOwogICAgICAgIG1hcmdpbjogMDsKICAgICAgICBwYWRkaW5nOiAwOwogICAgICAgIGZvbnQtZmFtaWx5OiAtYXBwbGUtc3lzdGVtLCBzeXN0ZW0tdWksIEJsaW5rTWFjU3lzdGVtRm9udCwgIlNlZ29lIFVJIiwgIk9wZW4gU2FucyIsICJIZWx2ZXRpY2EgTmV1ZSIsIEhlbHZldGljYSwgQXJpYWwsIHNhbnMtc2VyaWY7CiAgICAgICAgCiAgICB9CiAgICBkaXYgewogICAgICAgIHdpZHRoOiA2MDBweDsKICAgICAgICBtYXJnaW46IDVlbSBhdXRvOwogICAgICAgIHBhZGRpbmc6IDJlbTsKICAgICAgICBiYWNrZ3JvdW5kLWNvbG9yOiAjZmRmZGZmOwogICAgICAgIGJvcmRlci1yYWRpdXM6IDAuNWVtOwogICAgICAgIGJveC1zaGFkb3c6IDJweCAzcHggN3B4IDJweCByZ2JhKDAsMCwwLDAuMDIpOwogICAgfQogICAgYTpsaW5rLCBhOnZpc2l0ZWQgewogICAgICAgIGNvbG9yOiAjMzg0ODhmOwogICAgICAgIHRleHQtZGVjb3JhdGlvbjogbm9uZTsKICAgIH0KICAgIEBtZWRpYSAobWF4LXdpZHRoOiA3MDBweCkgewogICAgICAgIGRpdiB7CiAgICAgICAgICAgIG1hcmdpbjogMCBhdXRvOwogICAgICAgICAgICB3aWR0aDogYXV0bzsKICAgICAgICB9CiAgICB9CiAgICA8L3N0eWxlPiAgICAKPC9oZWFkPgoKPGJvZHk+CjxkaXY+CiAgICA8aDE+RXhhbXBsZSBEb21haW48L2gxPgogICAgPHA+VGhpcyBkb21haW4gaXMgZm9yIHVzZSBpbiBpbGx1c3RyYXRpdmUgZXhhbXBsZXMgaW4gZG9jdW1lbnRzLiBZb3UgbWF5IHVzZSB0aGlzCiAgICBkb21haW4gaW4gbGl0ZXJhdHVyZSB3aXRob3V0IHByaW9yIGNvb3JkaW5hdGlvbiBvciBhc2tpbmcgZm9yIHBlcm1pc3Npb24uPC9wPgogICAgPHA+PGEgaHJlZj0iaHR0cHM6Ly93d3cuaWFuYS5vcmcvZG9tYWlucy9leGFtcGxlIj5Nb3JlIGluZm9ybWF0aW9uLi4uPC9hPjwvcD4KPC9kaXY+CjwvYm9keT4KPC9odG1sPgo=",
        "ContentLength": -1,
        "TransferEncoding": [],
        "Trailer": null,
        "TLS": null,
        "Request": null
      },
      "ErrType": null,
      "ErrMsg": null,
      "UUID": "bf044888-81fd-43c0-a9cc-996accd9bf37"
    }
  ]
}`

	require.Equal(t, expected, got)
}
