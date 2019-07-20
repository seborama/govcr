package govcr

import "github.com/seborama/govcr/cassette"

// TrackMutator is an function signature for a Track mutator.
type TrackMutator func(*cassette.Track)

// OnErr adds a conditional mutation when the Track has recorded an error.
func (tm TrackMutator) OnErr() TrackMutator {
	return func(aTrack *cassette.Track) {
		if aTrack != nil && (aTrack.ErrType != "" || aTrack.ErrMsg != "") {
			tm(aTrack)
		}
		return
	}
}

// OnNoErr adds a conditional mutation when the Track has not recorded an error.
func (tm TrackMutator) OnNoErr() TrackMutator {
	return func(aTrack *cassette.Track) {
		if aTrack != nil && aTrack.ErrType == "" && aTrack.ErrMsg == "" {
			tm(aTrack)
		}
		return
	}
}

// TrackMutators is a collection of TrackMutator's.
type TrackMutators []TrackMutator

// Add a set of TrackMutator's to this TrackMutators collection.
func (tms TrackMutators) Add(mutators ...TrackMutator) TrackMutators {
	return append(tms, mutators...)
}

// Mutate applies all mutators in this TrackMutators collection to the specified track.
func (tms TrackMutators) Mutate(t *cassette.Track) {
	for _, tm := range tms {
		tm(t)
	}
}
