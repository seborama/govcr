package track

import (
	"regexp"
)

// Mutator is a function signature for a Track mutator.
// A Mutator can be used to mutate a track at recording or replaying time.
type Mutator func(*Track)

// On accepts a mutator only when the predicate is true.
func (tm Mutator) On(predicate Predicate) Mutator {
	return func(trk *Track) {
		if trk != nil && predicate(trk) {
			tm(trk)
		}
	}
}

// Any accepts a mutator when any the supplied predicate is true.
// See also the alias "Or".
// TODO: add tests.
func (tm Mutator) Any(predicates ...Predicate) Mutator {
	return func(trk *Track) {
		if trk != nil {
			return
		}

		for _, p := range predicates {
			if p(trk) {
				tm(trk)
				return
			}
		}
	}
}

// Or accepts a mutator when any the supplied predicate is true.
// It is an alias of "Any".
func (tm Mutator) Or(predicates ...Predicate) Mutator {
	return tm.Any(predicates...)
}

// Predicate is a function signature that takes a track.Track and returns a boolean.
// It is used to construct conditional mutators.
type Predicate func(trk *Track) bool

// OnErr accepts a mutator only when an (HTTP/net) error occurred.
func (tm Mutator) OnErr() Mutator {
	return tm.On(
		func(trk *Track) bool {
			return trk.ErrType != nil
		},
	)
}

// HasErr is a Predicate that returns true if the track records a transport error.
func HasErr() Predicate {
	return func(trk *Track) bool {
		return trk.ErrType == nil
	}
}

// HasNoErr is a Predicate that returns true if the track does not record a
// transport error.
func HasNoErr() Predicate {
	return func(trk *Track) bool {
		return trk.ErrType != nil
	}
}

// HasAnyMethod is a Predicate that returns true if the track Request method is one
// of the specified statuses.
func HasAnyMethod(methods ...string) Predicate {
	return func(trk *Track) bool {
		for _, m := range methods {
			if m == trk.Request.Method {
				return true
			}
		}
		return false
	}
}

// HasAnyStatus is a Predicate that returns true if the track Response HTTP status string
// is one of the specified statuses.
func HasAnyStatus(statuses ...string) Predicate {
	return func(trk *Track) bool {
		for _, c := range statuses {
			if trk.Response.Status == c {
				return true
			}
		}
		return false
	}
}

// HasAnyStatusCode is a Predicate that returns true if the track Response HTTP status code
// is one of the specified codes.
func HasAnyStatusCode(codes ...int) Predicate {
	return func(trk *Track) bool {
		for _, c := range codes {
			if trk.Response.StatusCode == c {
				return true
			}
		}
		return false
	}
}

// OnNoErr accepts a mutator only when no (HTTP/net) error occurred.
func (tm Mutator) OnNoErr() Mutator {
	return tm.On(HasErr())
}

// OnRequestMethod accepts a mutator only when the request method matches one of the specified methods.
// Methods are defined in Go's http package, e.g. http.MethodGet, ...
func (tm Mutator) OnRequestMethod(methods ...string) Mutator {
	return tm.On(HasAnyMethod(methods...))
}

// OnRequestPath accepts a mutator only when the request URL matches the specified path.
func (tm Mutator) OnRequestPath(pathRegEx string) Mutator {
	if pathRegEx == "" {
		pathRegEx = ".*"
	}

	re := regexp.MustCompile(pathRegEx)

	return tm.On(
		func(trk *Track) bool {
			return re.MatchString(trk.Request.URL.String())
		},
	)
}

// OnStatus accepts a mutator only when the response status matches one of the supplied statuses.
// Standard HTTP statuses are defined in Go's http package. See http.StatusText.
func (tm Mutator) OnStatus(statuses ...string) Mutator {
	return tm.On(HasAnyStatus(statuses...))
}

// OnStatusCode accepts a mutator only when the response status matches one of the supplied statuses.
// Status codes are defined in Go's http package, e.g. http.StatusOK, ...
func (tm Mutator) OnStatusCode(codes ...int) Mutator {
	return tm.On(HasAnyStatusCode(codes...))
}

// RequestAddHeaderValue adds or overwrites a header key / value to the request.
func RequestAddHeaderValue(key, value string) Mutator {
	return func(trk *Track) {
		if trk != nil {
			trk.Request.Header.Add(key, value)
		}
	}
}

// RequestDeleteHeaderKeys deletes one or more header keys from the request.
func RequestDeleteHeaderKeys(keys ...string) Mutator {
	return func(trk *Track) {
		if trk != nil {
			for _, key := range keys {
				trk.Request.Header.Del(key)
			}
		}
	}
}

// ResponseAddHeaderValue adds or overwrites a header key / value to the response.
func ResponseAddHeaderValue(key, value string) Mutator {
	return func(trk *Track) {
		if trk != nil {
			trk.Response.Header.Add(key, value)
		}
	}
}

// ResponseDeleteHeaderKeys deletes one or more header keys from the response.
func ResponseDeleteHeaderKeys(keys ...string) Mutator {
	return func(trk *Track) {
		if trk != nil {
			for _, key := range keys {
				trk.Response.Header.Del(key)
			}
		}
	}
}

// RequestTransferHeaderKeys transfers one or more headers from the response to the request.
func RequestTransferHeaderKeys(keys ...string) Mutator {
	return func(trk *Track) {
		if trk != nil {
			for _, key := range keys {
				trk.Request.Header.Add(key, trk.Response.Header.Get(key))
			}
		}
	}
}

// ResponseTransferHeaderKeys transfers one or more headers from the request to the response.
func ResponseTransferHeaderKeys(keys ...string) Mutator {
	return func(trk *Track) {
		if trk != nil {
			for _, key := range keys {
				trk.Response.Header.Add(key, trk.Request.Header.Get(key))
			}
		}
	}
}

// RequestTransferTrailerKeys transfers one or more trailers from the response to the request.
func RequestTransferTrailerKeys(keys ...string) Mutator {
	return func(trk *Track) {
		if trk != nil {
			for _, key := range keys {
				trk.Request.Trailer.Add(key, trk.Response.Trailer.Get(key))
			}
		}
	}
}

// ResponseTransferTrailerKeys transfers one or more trailers from the request to the response.
func ResponseTransferTrailerKeys(keys ...string) Mutator {
	return func(trk *Track) {
		if trk != nil {
			for _, key := range keys {
				trk.Response.Trailer.Add(key, trk.Request.Trailer.Get(key))
			}
		}
	}
}

// RequestChangeBody allows to change the body of the request.
// Supply a function that does input to output transformation.
func RequestChangeBody(fn func(b []byte) []byte) Mutator {
	return func(trk *Track) {
		if trk != nil {
			trk.Request.Body = fn(trk.Request.Body)
		}
	}
}

// ResponseChangeBody allows to change the body of the response.
// Supply a function that does input to output transformation.
func ResponseChangeBody(fn func(b []byte) []byte) Mutator {
	return func(trk *Track) {
		if trk != nil {
			trk.Response.Body = fn(trk.Response.Body)
		}
	}
}

// ResponseDeleteTLS removes TLS data from the response.
func ResponseDeleteTLS() Mutator {
	return func(trk *Track) {
		if trk != nil {
			trk.Response.TLS = nil
		}
	}
}

// Mutators is a collection of Track Mutator's.
type Mutators []Mutator

// Add a set of TrackMutator's to this TrackMutators collection.
func (tms Mutators) Add(mutators ...Mutator) Mutators {
	return append(tms, mutators...)
}

// Mutate applies all mutators in this Mutators collection to the specified Track.
func (tms Mutators) Mutate(trk *Track) {
	for _, tm := range tms {
		tm(trk)
	}
}
