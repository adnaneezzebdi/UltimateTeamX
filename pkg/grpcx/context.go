package grpcx

// Chiavi condivise per passare l'identita' utente tra servizi gRPC.
type contextKey string

// ContextUserIDKey definisce la chiave per il context locale (non gRPC).
const ContextUserIDKey contextKey = "user_id"

// UserIDMetadataKey definisce la chiave metadata per l'user_id su gRPC.
const UserIDMetadataKey = "user_id"
