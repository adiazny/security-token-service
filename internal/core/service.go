package core

/*
	Ports
*/

type Subject struct {
	Token string
}

type Token struct{}

type TokenExchanger interface {
	Exchange(subject Subject) (Token, error)
}

type TokenService struct {
}

func NewTokenService() *TokenService {
	return &TokenService{}
}
