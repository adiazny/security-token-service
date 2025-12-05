package core

/*
	Ports
*/

type Subject struct {
	Token string
}

type Token struct{}

type SecurityTokenService interface {
	Exchange(subject Subject) (Token, error)
}
