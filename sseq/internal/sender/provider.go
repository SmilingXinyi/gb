package sender

// PayloadWriter delivers flushed CLEF batches to a single backend.
type PayloadWriter interface {
	WritePayload(payload []byte)
	Close() error
}
