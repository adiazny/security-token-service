package core_test

import (
	"testing"

	"github.com/adiazny/security-token-service/internal/core"
	"github.com/stretchr/testify/assert"
)

func TestNewTokenService(t *testing.T) {
	t.Parallel()

	got := core.NewTokenService()

	assert.NotNil(t, got)
}
