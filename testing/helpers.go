package testing

// ChanWriter is a writer that writes to a channel
type ChanWriter struct {
	C chan []byte
}

// NewChanWriter creates a new channel writer
func NewChanWriter() *ChanWriter {
	return &ChanWriter{make(chan []byte)}
}

// Write writes to the channel
func (w *ChanWriter) Write(p []byte) (int, error) {
	d := make([]byte, len(p))
	copy(d, p)
	w.C <- d
	return len(p), nil
}
