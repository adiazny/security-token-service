package core

/*
	Ports
*/

type TokenRequest struct {
	SubjectToken     string
	SubjectTokenType string
}

type TokenResponse struct{}

type TokenExchanger interface {
	Exchange(tokenRequest TokenRequest) (TokenResponse, error)
}

type TokenService struct {
}

func NewTokenService() *TokenService {
	return &TokenService{}
}
