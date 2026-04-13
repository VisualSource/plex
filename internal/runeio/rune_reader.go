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
	rd.peekInvalid = false

	return r, utf8.RuneLen(r), nil
}

// Peek returns the next n runes without advancing the reader.
// If EOF is encountered, the available runes up to EOF are returned.
// After calling Peek, UnreadRune is disabled until the next Read,
// ReadRune, or Discard call.
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
	rd.peekInvalid = false

	return nil
}

// Forget discards all buffered runes before the current position, freeing
// memory. After calling Forget, UnreadRune will return ErrInvalidUnreadRune
// since the history before the current position is gone.
func (rd *RuneReader) Forget() {
	// Keep only the unread portion of the buffer (from current onwards).
	remaining := len(rd.buffer) - rd.current
	if remaining > 0 {
		copy(rd.buffer, rd.buffer[rd.current:])
	}
	rd.buffer = rd.buffer[:remaining]
	rd.current = 0
	rd.peekInvalid = true
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
