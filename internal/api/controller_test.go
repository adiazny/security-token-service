package api

import (
	"testing"

	"github.com/adiazny/security-token-service/internal/core"
	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/require"
)

type mockSecurityTokenService struct{}

func (m *mockSecurityTokenService) Exchange(tokenRequest core.TokenRequest) (core.TokenResponse, error) {
	return core.TokenResponse{}, nil
}

func TestNewController(t *testing.T) {
	t.Parallel()

	got := NewController(fiber.New(), &mockSecurityTokenService{})

	require.NotEmpty(t, got)
}
