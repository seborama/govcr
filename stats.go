package govcr

// Stats holds information about the cassette and
// VCR runtime.
type Stats struct {
	// TracksLoaded is the number of tracks that were loaded from the cassette.
	TracksLoaded int32

	// TracksRecorded is the number of new tracks recorded by VCR.
	TracksRecorded int32

	// TracksPlayed is the number of tracks played back straight from the cassette.
	// I.e. tracks that were already present on the cassette and were played back.
	TracksPlayed int32
}
