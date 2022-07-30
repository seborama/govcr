package govcr

import (
	"regexp"

	"github.com/seborama/govcr/v5/cassette/track"
)

// TrackMutator is a function signature for a Track mutator.
// A TrackMutator can be used to mutate a track at recording or replaying time.
type TrackMutator func(*track.Track)

// On accepts a mutator only when the predicate is true.
func (tm TrackMutator) On(predicate func(trk *track.Track) bool) TrackMutator {
	return func(trk *track.Track) {
		if trk != nil && predicate(trk) {
			tm(trk)
		}
	}
}

// OnErr accepts a mutator only when an (HTTP/net) error occurred.
func (tm TrackMutator) OnErr() TrackMutator {
	return tm.On(
		func(trk *track.Track) bool {
			return trk.ErrType != "" || trk.ErrMsg != ""
		},
	)
}

// OnNoErr accepts a mutator only when no (HTTP/net) error occurred.
func (tm TrackMutator) OnNoErr() TrackMutator {
	return tm.On(
		func(trk *track.Track) bool {
			return trk.ErrType == "" && trk.ErrMsg == ""
		},
	)
}

// OnRequestMethod accepts a mutator only when the request method matches one of the specified methods.
// Methods are defined in Go's http package, e.g. http.MethodGet, ...
func (tm TrackMutator) OnRequestMethod(methods ...string) TrackMutator {
	return tm.On(
		func(trk *track.Track) bool {
			for _, m := range methods {
				if m == trk.Request.Method {
					return true
				}
			}
			return false
		},
	)
}

// OnRequestPath accepts a mutator only when the request URL matches the specified path.
func (tm TrackMutator) OnRequestPath(pathRegEx string) TrackMutator {
	if pathRegEx == "" {
		pathRegEx = ".*"
	}
	re := regexp.MustCompile(pathRegEx)

	return tm.On(
		func(trk *track.Track) bool {
			return re.MatchString(trk.Request.URL.String())
		},
	)
}

// OnStatus accepts a mutator only when the response status matches one of the supplied statuses.
func (tm TrackMutator) OnStatus(statuses ...int) TrackMutator {
	return tm.On(
		func(trk *track.Track) bool {
			for _, s := range statuses {
				if trk.Response.StatusCode == s {
					return true
				}
			}
			return false
		},
	)
}

// OnStatusCode accepts a mutator only when the response status matches one of the supplied statuses.
// Status codes are defined in Go's http package, e.g. http.StatusOK, ...
func (tm TrackMutator) OnStatusCode(statuses ...int) TrackMutator {
	return tm.On(
		func(trk *track.Track) bool {
			for _, s := range statuses {
				if trk.Response.StatusCode == s {
					return true
				}
			}
			return false
		},
	)
}

// RequestAddHeaderValue adds or overwrites a header key / value to the request.
func RequestAddHeaderValue(key, value string) TrackMutator {
	return func(trk *track.Track) {
		if trk != nil {
			trk.Request.Header.Add(key, value)
		}
	}
}

// RequestDeleteHeaderKeys deletes one or more header keys from the request.
func RequestDeleteHeaderKeys(keys ...string) TrackMutator {
	return func(trk *track.Track) {
		if trk != nil {
			for _, key := range keys {
				trk.Request.Header.Del(key)
			}
		}
	}
}

// ResponseAddHeaderValue adds or overwrites a header key / value to the response.
func ResponseAddHeaderValue(key, value string) TrackMutator {
	return func(trk *track.Track) {
		if trk != nil {
			trk.Response.Header.Add(key, value)
		}
	}
}

// ResponseDeleteHeaderKeys deletes one or more header keys from the response.
func ResponseDeleteHeaderKeys(keys ...string) TrackMutator {
	return func(trk *track.Track) {
		if trk != nil {
			for _, key := range keys {
				trk.Response.Header.Del(key)
			}
		}
	}
}

// RequestTransferHeaderKeys transfers one or more headers from the response to the request.
func RequestTransferHeaderKeys(keys ...string) TrackMutator {
	return func(trk *track.Track) {
		if trk != nil {
			for _, key := range keys {
				trk.Request.Header.Add(key, trk.Response.Header.Get(key))
			}
		}
	}
}

// ResponseTransferHeaderKeys transfers one or more headers from the request to the response.
func ResponseTransferHeaderKeys(keys ...string) TrackMutator {
	return func(trk *track.Track) {
		if trk != nil {
			for _, key := range keys {
				trk.Response.Header.Add(key, trk.Request.Header.Get(key))
			}
		}
	}
}

// RequestTransferTrailerKeys transfers one or more trailers from the response to the request.
func RequestTransferTrailerKeys(keys ...string) TrackMutator {
	return func(trk *track.Track) {
		if trk != nil {
			for _, key := range keys {
				trk.Request.Trailer.Add(key, trk.Response.Trailer.Get(key))
			}
		}
	}
}

// ResponseTransferTrailerKeys transfers one or more trailers from the request to the response.
func ResponseTransferTrailerKeys(keys ...string) TrackMutator {
	return func(trk *track.Track) {
		if trk != nil {
			for _, key := range keys {
				trk.Response.Trailer.Add(key, trk.Request.Trailer.Get(key))
			}
		}
	}
}

// RequestChangeBody allows to change the body of the request.
// Supply a function that does input to output transformation.
func RequestChangeBody(fn func(b []byte) []byte) TrackMutator {
	return func(trk *track.Track) {
		if trk != nil {
			trk.Request.Body = fn(trk.Request.Body)
		}
	}
}

// ResponseChangeBody allows to change the body of the response.
// Supply a function that does input to output transformation.
func ResponseChangeBody(fn func(b []byte) []byte) TrackMutator {
	return func(trk *track.Track) {
		if trk != nil {
			trk.Response.Body = fn(trk.Response.Body)
		}
	}
}

// TrackMutators is a collection of TrackMutator's.
type TrackMutators []TrackMutator

// Add a set of TrackMutator's to this TrackMutators collection.
func (tms TrackMutators) Add(mutators ...TrackMutator) TrackMutators {
	return append(tms, mutators...)
}

// Mutate applies all mutators in this TrackMutators collection to the specified track.
func (tms TrackMutators) Mutate(t *track.Track) {
	for _, tm := range tms {
		tm(t)
	}
}
