package track

import (
	"net/http"
	"regexp"
)

// Predicate is a function signature that takes a Track and returns a boolean.
// It is used to construct conditional mutators.
type Predicate func(*Track) bool

// Any accepts one or more predicates and returns a new predicate that will evaluate
// to true when any the supplied predicate is true, otherwise false.
func Any(predicates ...Predicate) Predicate {
	return Predicate(
		func(trk *Track) bool {
			for _, p := range predicates {
				if p(trk) {
					return true
				}
			}

			return false
		},
	)
}

// All accepts one or more predicates and returns a new predicate that will evaluate
// to true when every of the supplied predicate is true, otherwise false.
func All(predicates ...Predicate) Predicate {
	return Predicate(
		func(trk *Track) bool {
			for _, p := range predicates {
				if !p(trk) {
					return false
				}
			}

			return true
		},
	)
}

// None requires all predicates to be false.
// I.e. it is the equivalent of Not(Any(...)).
func None(predicates ...Predicate) Predicate {
	return Not(Any(predicates...))
}

// Not accepts one predicate and returns its logically contrary evaluation.
// I.e. it returns true when the supplied predicate is false and vice-versa.
func Not(predicate Predicate) Predicate {
	return Predicate(
		func(trk *Track) bool {
			return !predicate(trk)
		},
	)
}

// Mutator is a function signature for a Track mutator.
// A Mutator can be used to mutate a track at recording or replaying time.
//
// When recording, Response.Request will be nil since the track already records the Request in
// its own track.Request object.
//
// When replaying (and _only_ just when _replaying_), Response.Request will be populated with
// the _current_ HTTP request.
type Mutator func(trk *Track)

// On accepts a mutator only when the predicate is true.
// On will cowardly avoid the case when trk is nil.
func (tm Mutator) On(predicate Predicate) Mutator {
	return func(trk *Track) {
		if trk != nil && predicate(trk) {
			tm(trk)
		}
	}
}

// Predicate is a function signature that takes a track.Track and returns a boolean.
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

// TrackRequestAddHeaderValue adds or overwrites a header key / value to the HTTP request.
func TrackRequestAddHeaderValue(key, value string) Mutator {
	return func(trk *Track) {
		if trk != nil {
			// TODO: add a debug log on trk.Response.Request != nil as it indicates replaying time rather than recording time.
			if trk.Request.Header == nil {
				trk.Request.Header = http.Header{}
			}
			trk.Request.Header.Add(key, value)
		}
	}
}

// TrackRequestDeleteHeaderKeys deletes one or more header keys from the track request.
// This is useful with a recording track mutator.
func TrackRequestDeleteHeaderKeys(keys ...string) Mutator {
	return func(trk *Track) {
		if trk != nil {
			// TODO: add a debug log on trk.Response.Request != nil as it indicates replaying time rather than recording time.
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
			if trk.Response.Header == nil {
				trk.Response.Header = http.Header{}
			}
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

// ResponseTransferHTTPHeaderKeys transfers one or more headers from the "current" Response.Request to the track response.
// This is _only_ useful with a replaying track mutator.
func ResponseTransferHTTPHeaderKeys(keys ...string) Mutator {
	return func(trk *Track) {
		if trk == nil {
			return
		}

		if trk.Response == nil {
			// TODO: add debug logging that this mutator was likely called at recording time or that it was called
			// on replaying a track that does not have a response (presumably a transport error occurred).
			return
		}

		if trk.Response.Request == nil {
			// TODO: add debug logging that this mutator was likely called at recording time and it is not correct usage.
			return
		}

		for _, key := range keys {
			// only transfer headers that actually exist
			if trk.Response.Request.Header.Values(key) != nil {
				// this test must be inside the loop so we only add a blank header when we know
				// we're going to populate it, otherwise retain the "nil" value untouched.
				if trk.Response.Header == nil {
					trk.Response.Header = http.Header{}
				}

				trk.Response.Header.Add(key, trk.Response.Request.Header.Get(key))
			}
		}
	}
}

// ResponseTransferHTTPTrailerKeys transfers one or more trailers from the HTTP request to the track response.
// This is _only_ useful with a replaying track mutator.
func ResponseTransferHTTPTrailerKeys(keys ...string) Mutator {
	return func(trk *Track) {
		if trk == nil {
			return
		}

		if trk.Response == nil {
			// TODO: add debug logging that this mutator was likely called at recording time or that it was called
			// on replaying a track that does not have a response (presumably a transport error occurred).
			return
		}

		if trk.Response.Request == nil {
			// TODO: add debug logging that this mutator was likely called at recording time and it is not correct usage.
			return
		}

		for _, key := range keys {
			// only transfer trailers that actually exist
			if trk.Response.Request.Trailer.Values(key) != nil {
				// this test must be inside the loop so we only add a blank trailer when we know
				// we're going to populate it, otherwise retain the "nil" value untouched.
				if trk.Response.Trailer == nil {
					trk.Response.Trailer = http.Header{}
				}

				trk.Response.Trailer.Add(key, trk.Response.Request.Trailer.Get(key))
			}
		}
	}
}

// TrackRequestChangeBody allows to change the body of the request.
// Supply a function that does input to output transformation.
// This is useful with a recording track mutator.
func TrackRequestChangeBody(fn func(b []byte) []byte) Mutator {
	return func(trk *Track) {
		if trk != nil {
			// TODO: add a debug log on trk.Response.Request != nil as it indicates replaying time rather than recording time.
			trk.Request.Body = fn(trk.Request.Body)
		}
	}
}

// ResponseChangeBody allows to change the body of the response.
// Supply a function that does input to output transformation.
func ResponseChangeBody(fn func(b []byte) []byte) Mutator {
	return func(trk *Track) {
		if trk != nil && trk.Response != nil {
			trk.Response.Body = fn(trk.Response.Body)
		}
	}
}

// ResponseDeleteTLS removes TLS data from the response.
func ResponseDeleteTLS() Mutator {
	return func(trk *Track) {
		if trk != nil && trk.Response != nil {
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
// Reminder that trk.Response.Request is nil at recording time and only populated
// at replaying time.
// See Mutator and Track.Response.Request for further details.
func (tms Mutators) Mutate(trk *Track) {
	for _, tm := range tms {
		tm(trk)
	}
}
