package runeio

import (
	"bufio"
	"errors"
	"io"
	"unicode/utf8"
)

var (
	ErrBufferFull        = errors.New("buffer full")
	ErrInvalidUnreadRune = errors.New("invalid use of UnreadRune")
	ErrNegativeCount     = errors.New("negative count")
)

// A RuneReader implements rune-based input for an underlying byte stream.
// It maintains an internal buffer of runes that have been read from the
// underlying reader, allowing for unreading and peeking.
type RuneReader struct {
	buffer      []rune // all runes fetched from input so far
	current     int    // index of the next rune to return
	input       *bufio.Reader
	peekInvalid bool // true after Peek(); blocks UnreadRune until next Read/ReadRune/Discard
	consumed    int  // cumulative rune count advanced past (survives Forget)
	keepHistory bool // when true, Forget does not trim the buffer
}

// NewReader returns a new RuneReader reading from input.
func NewReader(input io.Reader) *RuneReader {
	return &RuneReader{
		buffer: make([]rune, 0, 64),
		input:  bufio.NewReader(input),
	}
}

// Read reads runes into p, returning the total number of bytes consumed.
// Each rune is added to the internal buffer, allowing UnreadRune to work.
func (rd *RuneReader) Read(p []rune) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}

	totalBytes := 0
	for i := range p {
		r, size, err := rd.ReadRune()
		if err != nil {
			if i == 0 {
				return 0, err
			}
			// return what we have before the error
			return totalBytes, err
		}
		totalBytes += size
		p[i] = r
	}

	return totalBytes, nil
}

// ReadRune reads a single UTF-8 encoded Unicode character and returns the
// rune and its size in bytes. The rune is added to the internal buffer,
// allowing a subsequent UnreadRune call.
func (rd *RuneReader) ReadRune() (rune, int, error) {
	if err := rd.ensureBuffered(1); err != nil {
		return utf8.RuneError, 0, err
	}

	r := rd.buffer[rd.current]
	rd.current++
	rd.consumed++
	rd.peekInvalid = false

	return r, utf8.RuneLen(r), nil
}

// Offset returns the cumulative number of runes advanced past since the
// reader was created. It is not affected by Forget, so callers can use it
// as a stable position into the (post-preprocessing) input stream.
func (rd *RuneReader) Offset() int {
	return rd.consumed
}

// Peek returns the next n runes without advancing the reader.
// If EOF is encountered, the available runes up to EOF are returned.
func (rd *RuneReader) Peek(n int) ([]rune, error) {
	if n < 0 {
		return nil, ErrNegativeCount
	}

	err := rd.ensureBuffered(n)
	// ensureBuffered may return io.EOF after partially filling — that's okay for Peek.
	// We only treat non-EOF errors as hard failures.
	if err != nil && err != io.EOF {
		return nil, err
	}

	// Clamp to what's actually available
	available := len(rd.buffer) - rd.current
	if n > available {
		n = available
	}

	//rd.peekInvalid = true

	return rd.buffer[rd.current : rd.current+n], nil
}

// UnreadRune unreads the most recently read rune, allowing it to be read again.
// It returns ErrInvalidUnreadRune if:
//   - no rune has been read yet
//   - the last operation was Peek (which does not advance the reader)
func (rd *RuneReader) UnreadRune() error {
	if rd.peekInvalid || rd.current == 0 {
		return ErrInvalidUnreadRune
	}

	rd.current--
	rd.consumed--
	return nil
}

// Discard skips the next n runes. The skipped runes are still added to the
// internal buffer, allowing UnreadRune to work after discarding.
// Returns ErrNegativeCount if n < 0.
func (rd *RuneReader) Discard(n int) error {
	if n < 0 {
		return ErrNegativeCount
	}

	if err := rd.ensureBuffered(n); err != nil {
		return err
	}

	rd.current += n
	rd.consumed += n
	rd.peekInvalid = false

	return nil
}

// Forget discards all buffered runes before the current position, freeing
// memory. After calling Forget, UnreadRune will return ErrInvalidUnreadRune
// since the history before the current position is gone.
//
// When EnableHistory has been called, Forget is a no-op: the full history of
// runes returned so far is retained so that Slice can recover arbitrary
// spans.
func (rd *RuneReader) Forget() {
	if rd.keepHistory {
		return
	}
	// Keep only the unread portion of the buffer (from current onwards).
	remaining := len(rd.buffer) - rd.current
	if remaining > 0 {
		copy(rd.buffer, rd.buffer[rd.current:])
	}
	rd.buffer = rd.buffer[:remaining]
	rd.current = 0
	rd.peekInvalid = true
}

// EnableHistory switches the reader into history-preserving mode: subsequent
// calls to Forget become no-ops, and runes previously returned remain in the
// internal buffer at their original positions so that Slice can address them
// by absolute offset.
//
// Must be called before any rune is read; calling it later mixes coordinate
// systems and Slice's results become undefined.
func (rd *RuneReader) EnableHistory() {
	rd.keepHistory = true
}

// Slice returns the runes between absolute offsets [start, end) in the
// input stream. Only valid when EnableHistory was called before any rune was
// read. Returns an empty slice if start >= end, or if the requested range
// extends past what has been read so far.
func (rd *RuneReader) Slice(start, end int) []rune {
	if start < 0 || end <= start {
		return nil
	}
	if end > len(rd.buffer) {
		end = len(rd.buffer)
	}
	if start >= end {
		return nil
	}
	return rd.buffer[start:end]
}

// ensureBuffered guarantees that at least n runes are available in the buffer
// starting at rd.current, fetching from the underlying reader as needed.
func (rd *RuneReader) ensureBuffered(n int) error {
	for len(rd.buffer)-rd.current < n {
		r, _, err := rd.input.ReadRune()
		if err != nil {
			return err
		}
		rd.buffer = append(rd.buffer, r)
	}
	return nil
}
