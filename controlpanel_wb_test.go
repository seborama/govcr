package govcr

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/seborama/govcr/v8/cassette/track"
)

func TestControlPanel_SetRecordingMutators(t *testing.T) {
	unit := &ControlPanel{
		client: &http.Client{
			Transport: &vcrTransport{
				pcb: &PrintedCircuitBoard{
					trackRecordingMutators: track.Mutators{
						track.TrackRequestAddHeaderValue("k", "v"),
						track.TrackRequestDeleteHeaderKeys("k2"),
					},
				},
			},
		},
	}

	unit.SetRecordingMutators(track.TrackRequestDeleteHeaderKeys("k1"))

	assert.Len(t, unit.client.Transport.(*vcrTransport).pcb.trackRecordingMutators, 1)
	assert.Len(t, unit.client.Transport.(*vcrTransport).pcb.trackReplayingMutators, 0)
}

func TestControlPanel_AddRecordingMutators(t *testing.T) {
	unit := &ControlPanel{
		client: &http.Client{
			Transport: &vcrTransport{
				pcb: &PrintedCircuitBoard{
					trackRecordingMutators: track.Mutators{
						track.TrackRequestAddHeaderValue("k", "v"),
					},
				},
			},
		},
	}

	unit.AddRecordingMutators(track.TrackRequestDeleteHeaderKeys("k2"))

	assert.Len(t, unit.client.Transport.(*vcrTransport).pcb.trackRecordingMutators, 2)
	assert.Len(t, unit.client.Transport.(*vcrTransport).pcb.trackReplayingMutators, 0)
}

func TestControlPanel_SetReplayingMutators(t *testing.T) {
	unit := &ControlPanel{
		client: &http.Client{
			Transport: &vcrTransport{
				pcb: &PrintedCircuitBoard{
					trackReplayingMutators: track.Mutators{
						track.TrackRequestAddHeaderValue("k", "v"),
						track.TrackRequestDeleteHeaderKeys("k2"),
					},
				},
			},
		},
	}

	unit.SetReplayingMutators(track.TrackRequestDeleteHeaderKeys("k1"))

	assert.Len(t, unit.client.Transport.(*vcrTransport).pcb.trackReplayingMutators, 1)
	assert.Len(t, unit.client.Transport.(*vcrTransport).pcb.trackRecordingMutators, 0)
}

func TestControlPanel_AddReplayingMutators(t *testing.T) {
	unit := &ControlPanel{
		client: &http.Client{
			Transport: &vcrTransport{
				pcb: &PrintedCircuitBoard{
					trackReplayingMutators: track.Mutators{
						track.TrackRequestAddHeaderValue("k", "v"),
					},
				},
			},
		},
	}

	unit.AddReplayingMutators(track.TrackRequestDeleteHeaderKeys("k2"))

	assert.Len(t, unit.client.Transport.(*vcrTransport).pcb.trackReplayingMutators, 2)
	assert.Len(t, unit.client.Transport.(*vcrTransport).pcb.trackRecordingMutators, 0)
}
