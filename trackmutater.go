package govcr

type TrackRecordingMutater interface {
	Mutate(*Track)
}

type TrackRecordingMutaterFunc func(*Track)

func (m TrackRecordingMutaterFunc) Mutate(t *Track) {
	m(t)
}

func (m TrackRecordingMutaterFunc) OnErr() TrackRecordingMutaterFunc {
	return TrackRecordingMutaterFunc(func(t *Track) {
		if t != nil && (t.ErrType != "" || t.ErrMsg != "") {
			m.Mutate(t)
		}
	})
}

func (m TrackRecordingMutaterFunc) OnNoErr() TrackRecordingMutaterFunc {
	return TrackRecordingMutaterFunc(func(t *Track) {
		if t != nil && t.ErrType == "" && t.ErrMsg == "" {
			m.Mutate(t)
		}
	})
}
