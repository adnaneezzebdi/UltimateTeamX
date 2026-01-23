package lock

import "github.com/google/uuid"

// newToken genera un token casuale per il lock.
func newToken() string {
	return uuid.NewString()
}
