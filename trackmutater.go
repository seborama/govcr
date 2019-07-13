package govcr

// TrackRecordingMutater is an interface that defines a method to mutate
// a Track.
type TrackRecordingMutater interface {
	Mutate(*Track)
}

// TrackRecordingMutaterFunc is an function signature for a Track mutator.
type TrackRecordingMutaterFunc func(*Track)

// Mutate implements TrackRecordingMutater for a Track recording mutator.
func (m TrackRecordingMutaterFunc) Mutate(t *Track) {
	m(t)
}

// OnErr adds a conditional mutation when the Track has recorded an error.
func (m TrackRecordingMutaterFunc) OnErr() TrackRecordingMutaterFunc {
	return TrackRecordingMutaterFunc(func(t *Track) {
		if t != nil && (t.ErrType != "" || t.ErrMsg != "") {
			m.Mutate(t)
		}
	})
}

// OnNoErr adds a conditional mutation when the Track has not recorded an error.
func (m TrackRecordingMutaterFunc) OnNoErr() TrackRecordingMutaterFunc {
	return TrackRecordingMutaterFunc(func(t *Track) {
		if t != nil && t.ErrType == "" && t.ErrMsg == "" {
			m.Mutate(t)
		}
	})
}
