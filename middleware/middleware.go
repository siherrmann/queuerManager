package middleware

import (
	"crypto/rand"
)

type Middleware struct {
	csrfKey []byte
}

func NewMiddleware() *Middleware {
	csrfKey := make([]byte, 32)
	n, err := rand.Read(csrfKey)
	if err != nil {
		panic(err)
	}
	if n != 32 {
		panic("unable to read 32 bytes for CSRF key")
	}

	return &Middleware{
		csrfKey: csrfKey,
	}
}
