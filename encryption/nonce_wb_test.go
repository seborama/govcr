package encryption

import (
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_validateNonceGenerator_Passes(t *testing.T) {
	err := validateNonceGenerator(NewRandomNonceGenerator(32))
	assert.NoError(t, err)
}

func Test_validateNonceGenerator_Fails_NonceGenErr(t *testing.T) {
	err := validateNonceGenerator(brokenNonceGenerator{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "nonceGenerator failure: broken nonce")
}

func Test_validateNonceGenerator_Fails_WeakNonceGen(t *testing.T) {
	err := validateNonceGenerator(weakNonceGenerator{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "nonceGenerator produces frequent duplicates")
}

type brokenNonceGenerator struct{}

func (ng brokenNonceGenerator) Generate() ([]byte, error) {
	return nil, errors.New("broken nonce")
}

type weakNonceGenerator struct{}

func (ng weakNonceGenerator) Generate() ([]byte, error) {
	return []byte("static nonce"), nil
}
