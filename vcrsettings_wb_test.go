package govcr

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWithLiveOnlyMode(t *testing.T) {
	vcrSettings := &VCRSettings{}

	WithLiveOnlyMode()(vcrSettings)
	assert.Equal(t, HTTPModeLiveOnly, vcrSettings.httpMode)
}

func TestWithOfflineMode(t *testing.T) {
	vcrSettings := &VCRSettings{}

	WithOfflineMode()(vcrSettings)
	assert.Equal(t, HTTPModeOffline, vcrSettings.httpMode)
}

func TestW(t *testing.T) {
	vcrSettings := &VCRSettings{}

	WithReadOnlyMode()(vcrSettings)
	assert.True(t, vcrSettings.readOnly)
}
