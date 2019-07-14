package govcr

// TrackMutator is an function signature for a Track mutator.
type TrackMutator func(*Track) *Track

// OnErr adds a conditional mutation when the Track has recorded an error.
func (tm TrackMutator) OnErr() TrackMutator {
	return func(t *Track) *Track {
		if t != nil && (t.ErrType != "" || t.ErrMsg != "") {
			return tm(t)
		}
		return t
	}
}

// OnNoErr adds a conditional mutation when the Track has not recorded an error.
func (tm TrackMutator) OnNoErr() TrackMutator {
	return func(t *Track) *Track {
		if t != nil && t.ErrType == "" && t.ErrMsg == "" {
			return tm(t)
		}
		return t
	}
}

// TrackMutators is a collection of TrackMutator's.
type TrackMutators []TrackMutator

// Add a set of TrackMutator's to this TrackMutators collection.
func (tms TrackMutators) Add(mutators ...TrackMutator) TrackMutators {
	return append(tms, mutators...)
}

// Mutate applies all mutators in this TrackMutators collection to the specified track.
func (tms TrackMutators) Mutate(t *Track) {
	for _, mutator := range tms {
		t = mutator(t)
	}
}
