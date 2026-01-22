package club

import "errors"

// Errori di dominio usati da service/repo e mappati nel layer gRPC.
var ErrClubNotFound = errors.New("club not found")

// ErrUnauthenticated indica credenziali mancanti o invalide.
var ErrUnauthenticated = errors.New("unauthenticated")
