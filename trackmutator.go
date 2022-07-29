package govcr

import (
	"github.com/seborama/govcr/v5/cassette/track"
)

// TrackMutator is a function signature for a Track mutator.
// A TrackMutator can be used to mutate a track at recording or replaying time.
type TrackMutator func(*track.Track)

// OnErr adds a conditional mutation when the Track has recorded an error.
func (tm TrackMutator) OnErr() TrackMutator {
	return func(aTrack *track.Track) {
		if aTrack != nil && (aTrack.ErrType != "" || aTrack.ErrMsg != "") {
			tm(aTrack)
		}
	}
}

// OnNoErr adds a conditional mutation when the Track has not recorded an error.
func (tm TrackMutator) OnNoErr() TrackMutator {
	return func(aTrack *track.Track) {
		if aTrack != nil && aTrack.ErrType == "" && aTrack.ErrMsg == "" {
			tm(aTrack)
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
