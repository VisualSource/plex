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

// A Reader implements rune-based input for an underlying byte stream.
type RuneReader struct {
	buffer  []rune
	current int
	input   *bufio.Reader
}

// NewReader returns a new Reader reading the given input.
func NewReader(input io.Reader) *RuneReader {
	return &RuneReader{
		buffer: make([]rune, 1024),
		input:  bufio.NewReader(input),
	}
}

// Read a given number of runes.
func (rd *RuneReader) Read(p []rune) (int, error) {
	size := 0

	for i := 0; i < len(p); i++ {
		r, s, err := rd.ReadRune()
		if err != nil {
			return 0, err
		}

		size += s
		p[i] = r
	}

	return size, nil
}

// Reads a single UTF-8 encoded Unicode character and returns the rune and its size in bytes
func (rd *RuneReader) ReadRune() (rune, int, error) {
	if err := rd.fillBuffer(); err != nil {
		return utf8.RuneError, 0, err
	}

	r := rd.buffer[rd.current]
	rd.current++

	return r, utf8.RuneLen(r), nil
}

// Returnes n number of runes from buffer without advacing the reader.
// if EOF is encoundered only that number of runes will be returned
// this will prevent unreadRune call from being useable
func (rd *RuneReader) Peek(n int) ([]rune, error) {
	if n < 0 {
		return nil, ErrNegativeCount
	}

	if n > len(rd.buffer) {
		return nil, ErrBufferFull
	}

	//TODO update buff

	return rd.buffer[rd.current : rd.current+n], nil
}

// unreads the current read rune
func (rd *RuneReader) UnreadRune() error {
	if rd.current == 0 {
		return ErrInvalidUnreadRune
	}

	rd.current--
	return nil
}

// skip n number of runes
func (rd *RuneReader) Discard(n int) error {

	return nil
}

// Discards buffered runes before the current input position.
// will cause following unreadRune calls to fail
func (rd *RuneReader) Forget() {}

func (rd *RuneReader) fillBuffer() error {
	r, _, err := rd.input.ReadRune()
	if err != nil {
		return err
	}

	rd.buffer = append(rd.buffer, r)

	return nil
}
