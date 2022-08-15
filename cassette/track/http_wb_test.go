package track

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_cloneURLValues(t *testing.T) {
	unit := url.Values{
		"one": {
			"one.1", "one.2",
		},
		"two": {
			"two.1", "",
		},
		"three": {},
	}

	got := cloneURLValues(unit)
	assert.Equal(t, unit, got)
}
