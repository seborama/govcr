package govcr

import (
	"bytes"
	"crypto/tls"
	"net/http"
	"reflect"
	"strings"
	"testing"
)

func Test_trackReplaysError(t *testing.T) {
	t1 := track{
		ErrType:  "*net.OpError",
		ErrMsg:   "Some test error",
		Response: response{},
	}

	_, err := t1.response(&http.Request{})
	want := "govcr govcr: *net.OpError: Some test error"
	if err != nil && err.Error() != want {
		t.Errorf("got error '%s', want '%s'\n", err.Error(), want)
	}
}

func Test_cassette_gzipFilter(t *testing.T) {
	type fields struct {
		Name   string
		Path   string
		Tracks []track
		stats  Stats
	}
	type args struct {
		data bytes.Buffer
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    []byte
		wantErr bool
	}{
		{
			name: "Should not compress data",
			fields: fields{
				Name: "cassette",
			},
			args: args{
				data: *bytes.NewBufferString(`data`),
			},
			want:    []byte(`data`),
			wantErr: false,
		},
		{
			name: "Should compress data when cassette name is *.gz",
			fields: fields{
				Name: "cassette.gz",
			},
			args: args{
				data: *bytes.NewBufferString(`data`),
			},
			want:    []byte{31, 139, 8, 0, 0, 0, 0, 0, 0, 255, 74, 73, 44, 73, 4, 4, 0, 0, 255, 255, 99, 243, 243, 173, 4, 0, 0, 0},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			k7 := newCassette(tt.fields.Name, tt.fields.Path)
			k7.Tracks = tt.fields.Tracks
			k7.tracksLoaded = tt.fields.stats.TracksLoaded

			got, err := k7.gzipFilter(tt.args.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("cassette.gzipFilter() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("cassette.gzipFilter() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_cassette_isLongPlay(t *testing.T) {
	type fields struct {
		Name   string
		Path   string
		Tracks []track
		stats  Stats
	}
	tests := []struct {
		name   string
		fields fields
		want   bool
	}{
		{
			name: "Should detect Long Play cassette (i.e. compressed)",
			fields: fields{
				Name: "cassette.gz",
			},
			want: true,
		},
		{
			name: "Should detect Normal Play cassette (i.e. not compressed)",
			fields: fields{
				Name: "cassette",
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			k7 := newCassette(tt.fields.Name, tt.fields.Path)
			k7.Tracks = tt.fields.Tracks
			k7.tracksLoaded = tt.fields.stats.TracksLoaded

			if got := k7.isLongPlay(); got != tt.want {
				t.Errorf("cassette.isLongPlay() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_cassette_gunzipFilter(t *testing.T) {
	type fields struct {
		Name   string
		Path   string
		Tracks []track
		stats  Stats
	}
	type args struct {
		data []byte
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    []byte
		wantErr bool
	}{
		{
			name: "Should not compress data",
			fields: fields{
				Name: "cassette",
			},
			args: args{
				data: []byte(`data`),
			},
			want:    []byte(`data`),
			wantErr: false,
		},
		{
			name: "Should de-compress data when cassette name is *.gz",
			fields: fields{
				Name: "cassette.gz",
			},
			args: args{
				data: []byte{31, 139, 8, 0, 0, 0, 0, 0, 0, 255, 74, 73, 44, 73, 4, 4, 0, 0, 255, 255, 99, 243, 243, 173, 4, 0, 0, 0},
			},
			want:    []byte(`data`),
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			k7 := newCassette(tt.fields.Name, tt.fields.Path)
			k7.Tracks = tt.fields.Tracks
			k7.tracksLoaded = tt.fields.stats.TracksLoaded

			got, err := k7.gunzipFilter(tt.args.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("cassette.gunzipFilter() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("cassette.gunzipFilter() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_cassetteNameToFilename(t *testing.T) {
	type args struct {
		cassetteName string
		cassettePath string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "Should return normal cassette name",
			args: args{
				cassetteName: "MyCassette",
			},
			want: "MyCassette.cassette",
		},
		{
			name: "Should return normalised gz cassette name",
			args: args{
				cassetteName: "MyCassette.gz",
			},
			want: "MyCassette.cassette.gz",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := cassetteNameToFilename(tt.args.cassetteName, tt.args.cassettePath); !strings.HasSuffix(got, tt.want) {
				t.Errorf("cassetteNameToFilename() = %v, want suffix %v", got, tt.want)
			}
		})
	}
}

func Test_cassette_addTrack(t *testing.T) {
	type fields struct {
		removeTLS bool
	}
	type args struct {
		track track
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		{
			name: "with tls, keep",
			fields: fields{
				removeTLS: false,
			},
			args: args{
				track: track{
					Response: response{
						TLS: &tls.ConnectionState{},
					},
				},
			},
		},
		{
			name: "with tls, remove",
			fields: fields{
				removeTLS: true,
			},
			args: args{
				track: track{
					Response: response{
						TLS: &tls.ConnectionState{},
					},
				},
			},
		},
		{
			name: "without tls, keep",
			fields: fields{
				removeTLS: false,
			},
			args: args{
				track: track{
					Response: response{
						TLS: nil,
					},
				},
			},
		},
		{
			name: "without tls, remove",
			fields: fields{
				removeTLS: true,
			},
			args: args{
				track: track{
					Response: response{
						TLS: nil,
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			k7 := newCassette(tt.name, tt.name)
			k7.removeTLS = tt.fields.removeTLS

			k7.addTrack(&tt.args.track)
			gotTLS := k7.Tracks[0].Response.TLS != nil
			if gotTLS && tt.fields.removeTLS {
				t.Errorf("got TLS, but it should have been removed")
			}
			if !gotTLS && !tt.fields.removeTLS && tt.args.track.Response.TLS != nil {
				t.Errorf("tls was removed, but shouldn't")
			}
		})
	}
}
