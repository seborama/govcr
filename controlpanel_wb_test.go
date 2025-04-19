package govcr

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/seborama/govcr/v17/cassette/track"
)

func TestControlPanel_SetRecordingMutators(t *testing.T) {
	unit := &ControlPanel{
		client: &http.Client{
			Transport: &vcrTransport{
				pcb: &PrintedCircuitBoard{
					trackRecordingMutators: track.Mutators{
						track.AddTrackRequestHeaderValue("k", "v"),
						track.DeleteTrackRequestHeaderKeys("k2"),
					},
				},
			},
		},
	}

	unit.SetRecordingMutators(track.DeleteTrackRequestHeaderKeys("k1"))

	assert.Len(t, unit.client.Transport.(*vcrTransport).pcb.trackRecordingMutators, 1)
	assert.Empty(t, unit.client.Transport.(*vcrTransport).pcb.trackReplayingMutators)
}

func TestControlPanel_AddRecordingMutators(t *testing.T) {
	unit := &ControlPanel{
		client: &http.Client{
			Transport: &vcrTransport{
				pcb: &PrintedCircuitBoard{
					trackRecordingMutators: track.Mutators{
						track.AddTrackRequestHeaderValue("k", "v"),
					},
				},
			},
		},
	}

	unit.AddRecordingMutators(track.DeleteTrackRequestHeaderKeys("k2"))

	assert.Len(t, unit.client.Transport.(*vcrTransport).pcb.trackRecordingMutators, 2)
	assert.Empty(t, unit.client.Transport.(*vcrTransport).pcb.trackReplayingMutators)
}

func TestControlPanel_SetReplayingMutators(t *testing.T) {
	unit := &ControlPanel{
		client: &http.Client{
			Transport: &vcrTransport{
				pcb: &PrintedCircuitBoard{
					trackReplayingMutators: track.Mutators{
						track.AddTrackRequestHeaderValue("k", "v"),
						track.DeleteTrackRequestHeaderKeys("k2"),
					},
				},
			},
		},
	}

	unit.SetReplayingMutators(track.DeleteTrackRequestHeaderKeys("k1"))

	assert.Len(t, unit.client.Transport.(*vcrTransport).pcb.trackReplayingMutators, 1)
	assert.Empty(t, unit.client.Transport.(*vcrTransport).pcb.trackRecordingMutators)
}

func TestControlPanel_AddReplayingMutators(t *testing.T) {
	unit := &ControlPanel{
		client: &http.Client{
			Transport: &vcrTransport{
				pcb: &PrintedCircuitBoard{
					trackReplayingMutators: track.Mutators{
						track.AddTrackRequestHeaderValue("k", "v"),
					},
				},
			},
		},
	}

	unit.AddReplayingMutators(track.DeleteTrackRequestHeaderKeys("k2"))

	assert.Len(t, unit.client.Transport.(*vcrTransport).pcb.trackReplayingMutators, 2)
	assert.Empty(t, unit.client.Transport.(*vcrTransport).pcb.trackRecordingMutators)
}
