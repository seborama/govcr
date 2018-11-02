package govcr

import (
	"bytes"
	"reflect"
	"strings"
	"testing"
)

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
			k7 := &cassette{
				Name:   tt.fields.Name,
				Path:   tt.fields.Path,
				Tracks: tt.fields.Tracks,
				stats:  tt.fields.stats,
			}
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
			k7 := &cassette{
				Name:   tt.fields.Name,
				Path:   tt.fields.Path,
				Tracks: tt.fields.Tracks,
				stats:  tt.fields.stats,
			}
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
			k7 := &cassette{
				Name:   tt.fields.Name,
				Path:   tt.fields.Path,
				Tracks: tt.fields.Tracks,
				stats:  tt.fields.stats,
			}
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
