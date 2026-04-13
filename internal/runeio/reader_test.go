package runeio

import (
	"io"
	"strings"
	"testing"
	"unicode/utf8"
)

// helper: create a RuneReader from a string
func newReaderFromString(s string) *RuneReader {
	return NewReader(strings.NewReader(s))
}

// ---------------------------------------------------------------------------
// NewReader
// ---------------------------------------------------------------------------

func TestNewReader(t *testing.T) {
	rd := newReaderFromString("hello")
	if rd == nil {
		t.Fatal("NewReader returned nil")
	}
	if rd.input == nil {
		t.Fatal("NewReader did not initialise underlying bufio.Reader")
	}
	if rd.buffer == nil {
		t.Fatal("NewReader did not initialise buffer")
	}
}

// ---------------------------------------------------------------------------
// ReadRune
// ---------------------------------------------------------------------------

func TestReadRune_ASCII(t *testing.T) {
	rd := newReaderFromString("abc")

	for _, want := range []rune{'a', 'b', 'c'} {
		r, size, err := rd.ReadRune()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if r != want {
			t.Errorf("got rune %q, want %q", r, want)
		}
		if size != 1 {
			t.Errorf("got size %d, want 1", size)
		}
	}
}

func TestReadRune_Multibyte(t *testing.T) {
	// "日本語" — each rune is 3 bytes in UTF-8
	input := "日本語"
	rd := newReaderFromString(input)

	for _, want := range []rune{'日', '本', '語'} {
		r, size, err := rd.ReadRune()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if r != want {
			t.Errorf("got rune %q, want %q", r, want)
		}
		wantSize := utf8.RuneLen(want)
		if size != wantSize {
			t.Errorf("got size %d, want %d", size, wantSize)
		}
	}
}

func TestReadRune_EOF(t *testing.T) {
	rd := newReaderFromString("a")
	rd.ReadRune() // consume the only rune

	_, _, err := rd.ReadRune()
	if err != io.EOF {
		t.Errorf("expected io.EOF, got %v", err)
	}
}

func TestReadRune_Empty(t *testing.T) {
	rd := newReaderFromString("")
	_, _, err := rd.ReadRune()
	if err != io.EOF {
		t.Errorf("expected io.EOF on empty reader, got %v", err)
	}
}

func TestReadRune_BuffersRune(t *testing.T) {
	rd := newReaderFromString("ab")
	rd.ReadRune()
	rd.ReadRune()

	// Both runes should now be in the internal buffer
	if len(rd.buffer) < 2 {
		t.Errorf("expected at least 2 runes in buffer, got %d", len(rd.buffer))
	}
}

// ---------------------------------------------------------------------------
// Read
// ---------------------------------------------------------------------------

func TestRead_Full(t *testing.T) {
	rd := newReaderFromString("hello")
	p := make([]rune, 5)

	n, err := rd.Read(p)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n != 5 { // 5 ASCII bytes
		t.Errorf("got byte count %d, want 5", n)
	}
	if string(p) != "hello" {
		t.Errorf("got %q, want %q", string(p), "hello")
	}
}

func TestRead_Partial(t *testing.T) {
	rd := newReaderFromString("hello")
	p := make([]rune, 3)

	_, err := rd.Read(p)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(p) != "hel" {
		t.Errorf("got %q, want %q", string(p), "hel")
	}
}

func TestRead_Multibyte(t *testing.T) {
	input := "こんにちは" // 5 runes, each 3 bytes
	rd := newReaderFromString(input)
	p := make([]rune, 5)

	n, err := rd.Read(p)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n != 15 { // 5 runes × 3 bytes each
		t.Errorf("got byte count %d, want 15", n)
	}
	if string(p) != input {
		t.Errorf("got %q, want %q", string(p), input)
	}
}

func TestRead_EmptySlice(t *testing.T) {
	rd := newReaderFromString("hello")
	n, err := rd.Read([]rune{})
	if err != nil {
		t.Fatalf("unexpected error reading into empty slice: %v", err)
	}
	if n != 0 {
		t.Errorf("expected 0 bytes, got %d", n)
	}
}

func TestRead_PartialOnEOF(t *testing.T) {
	// Slice larger than input — should return what's available and then EOF
	rd := newReaderFromString("hi")
	p := make([]rune, 10)

	_, err := rd.Read(p)
	if err != io.EOF {
		t.Errorf("expected io.EOF when reading past end, got %v", err)
	}
	if p[0] != 'h' || p[1] != 'i' {
		t.Errorf("expected runes 'h','i' before EOF, got %q %q", p[0], p[1])
	}
}

func TestRead_BuffersRunes(t *testing.T) {
	rd := newReaderFromString("abc")
	p := make([]rune, 3)
	rd.Read(p)

	if len(rd.buffer) < 3 {
		t.Errorf("expected at least 3 runes in buffer after Read, got %d", len(rd.buffer))
	}
}

// ---------------------------------------------------------------------------
// UnreadRune
// ---------------------------------------------------------------------------

func TestUnreadRune_Basic(t *testing.T) {
	rd := newReaderFromString("ab")

	r1, _, _ := rd.ReadRune()
	if err := rd.UnreadRune(); err != nil {
		t.Fatalf("unexpected error on UnreadRune: %v", err)
	}

	r2, _, err := rd.ReadRune()
	if err != nil {
		t.Fatalf("unexpected error re-reading rune: %v", err)
	}
	if r1 != r2 {
		t.Errorf("unread rune mismatch: got %q, want %q", r2, r1)
	}
}

func TestUnreadRune_MultipleSequential(t *testing.T) {
	rd := newReaderFromString("abc")

	rd.ReadRune() // 'a'
	rd.ReadRune() // 'b'

	if err := rd.UnreadRune(); err != nil {
		t.Fatalf("first UnreadRune failed: %v", err)
	}

	r, _, err := rd.ReadRune()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r != 'b' {
		t.Errorf("expected 'b' after unread, got %q", r)
	}

	if err := rd.UnreadRune(); err != nil {
		t.Fatalf("second UnreadRune failed: %v", err)
	}

	r, _, err = rd.ReadRune()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r != 'b' {
		t.Errorf("expected 'b' after second unread, got %q", r)
	}
}

func TestUnreadRune_BeforeAnyRead(t *testing.T) {
	rd := newReaderFromString("abc")
	if err := rd.UnreadRune(); err != ErrInvalidUnreadRune {
		t.Errorf("expected ErrInvalidUnreadRune, got %v", err)
	}
}

func TestUnreadRune_AfterPeek(t *testing.T) {
	rd := newReaderFromString("abc")
	rd.Peek(2)

	if err := rd.UnreadRune(); err != ErrInvalidUnreadRune {
		t.Errorf("expected ErrInvalidUnreadRune after Peek, got %v", err)
	}
}

func TestUnreadRune_AfterForget(t *testing.T) {
	rd := newReaderFromString("abc")
	rd.ReadRune()
	rd.Forget()

	if err := rd.UnreadRune(); err != ErrInvalidUnreadRune {
		t.Errorf("expected ErrInvalidUnreadRune after Forget, got %v", err)
	}
}

func TestUnreadRune_ClearedAfterReadRune(t *testing.T) {
	// Peek disables UnreadRune; a subsequent ReadRune should re-enable it.
	rd := newReaderFromString("abc")
	rd.Peek(1)

	rd.ReadRune() // clears peekInvalid
	if err := rd.UnreadRune(); err != nil {
		t.Errorf("expected UnreadRune to work after ReadRune following Peek, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// Peek
// ---------------------------------------------------------------------------

func TestPeek_Basic(t *testing.T) {
	rd := newReaderFromString("hello")
	runes, err := rd.Peek(3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(runes) != "hel" {
		t.Errorf("got %q, want %q", string(runes), "hel")
	}
}

func TestPeek_DoesNotAdvance(t *testing.T) {
	rd := newReaderFromString("hello")
	rd.Peek(3)

	r, _, err := rd.ReadRune()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r != 'h' {
		t.Errorf("reader advanced after Peek; got %q, want 'h'", r)
	}
}

func TestPeek_MultipleCalls(t *testing.T) {
	rd := newReaderFromString("hello")
	r1, _ := rd.Peek(2)
	r2, _ := rd.Peek(2)

	if string(r1) != string(r2) {
		t.Errorf("repeated Peek returned different results: %q vs %q", string(r1), string(r2))
	}
}

func TestPeek_PastEOF(t *testing.T) {
	rd := newReaderFromString("hi")
	runes, err := rd.Peek(10)
	// err may be io.EOF but we should still get the available runes
	if err != nil && err != io.EOF {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(runes) != "hi" {
		t.Errorf("got %q, want %q", string(runes), "hi")
	}
}

func TestPeek_Zero(t *testing.T) {
	rd := newReaderFromString("hello")
	runes, err := rd.Peek(0)
	if err != nil {
		t.Fatalf("unexpected error on Peek(0): %v", err)
	}
	if len(runes) != 0 {
		t.Errorf("expected empty slice from Peek(0), got %v", runes)
	}
}

func TestPeek_Negative(t *testing.T) {
	rd := newReaderFromString("hello")
	_, err := rd.Peek(-1)
	if err != ErrNegativeCount {
		t.Errorf("expected ErrNegativeCount, got %v", err)
	}
}

func TestPeek_DisablesUnreadRune(t *testing.T) {
	rd := newReaderFromString("hello")
	rd.ReadRune()
	rd.Peek(2)

	if err := rd.UnreadRune(); err != nil {
		t.Errorf("expected ErrInvalidUnreadRune after Peek, got %v", err)
	}
}

func TestPeek_Multibyte(t *testing.T) {
	rd := newReaderFromString("日本語")
	runes, err := rd.Peek(2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(runes) != "日本" {
		t.Errorf("got %q, want %q", string(runes), "日本")
	}
}

// ---------------------------------------------------------------------------
// Discard
// ---------------------------------------------------------------------------

func TestDiscard_Basic(t *testing.T) {
	rd := newReaderFromString("hello")
	if err := rd.Discard(2); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	r, _, err := rd.ReadRune()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r != 'l' {
		t.Errorf("got %q after Discard(2), want 'l'", r)
	}
}

func TestDiscard_All(t *testing.T) {
	rd := newReaderFromString("hi")
	if err := rd.Discard(2); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, _, err := rd.ReadRune()
	if err != io.EOF {
		t.Errorf("expected EOF after discarding all runes, got %v", err)
	}
}

func TestDiscard_AllowsUnreadRune(t *testing.T) {
	rd := newReaderFromString("hello")
	rd.Discard(2)

	// UnreadRune should be valid since Discard buffers the runes
	if err := rd.UnreadRune(); err != nil {
		t.Errorf("expected UnreadRune to work after Discard, got %v", err)
	}

	// Re-reading should give us 'e', the last discarded rune
	r, _, err := rd.ReadRune()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r != 'e' {
		t.Errorf("expected 'e' after UnreadRune following Discard, got %q", r)
	}
}

func TestDiscard_Negative(t *testing.T) {
	rd := newReaderFromString("hello")
	if err := rd.Discard(-1); err != ErrNegativeCount {
		t.Errorf("expected ErrNegativeCount, got %v", err)
	}
}

func TestDiscard_Zero(t *testing.T) {
	rd := newReaderFromString("hello")
	if err := rd.Discard(0); err != nil {
		t.Errorf("unexpected error on Discard(0): %v", err)
	}

	r, _, _ := rd.ReadRune()
	if r != 'h' {
		t.Errorf("Discard(0) advanced reader; got %q, want 'h'", r)
	}
}

func TestDiscard_PastEOF(t *testing.T) {
	rd := newReaderFromString("hi")
	err := rd.Discard(10)
	if err != io.EOF {
		t.Errorf("expected io.EOF when discarding past end, got %v", err)
	}
}

func TestDiscard_ClearsPeekInvalid(t *testing.T) {
	rd := newReaderFromString("hello")
	rd.Peek(1)
	rd.Discard(1)

	// After Discard, UnreadRune should work again
	if err := rd.UnreadRune(); err != nil {
		t.Errorf("expected UnreadRune to work after Discard, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// Forget
// ---------------------------------------------------------------------------

func TestForget_Basic(t *testing.T) {
	rd := newReaderFromString("hello")
	rd.ReadRune() // 'h'
	rd.ReadRune() // 'e'
	rd.Forget()

	// current should be reset to 0
	if rd.current != 0 {
		t.Errorf("expected current == 0 after Forget, got %d", rd.current)
	}
}

func TestForget_DiscardsHistory(t *testing.T) {
	rd := newReaderFromString("hello")
	rd.ReadRune() // 'h'
	rd.ReadRune() // 'e'
	rd.Forget()

	// The buffer should only contain unread runes ('l','l','o' not yet fetched,
	// and nothing before current position)
	if rd.current != 0 {
		t.Errorf("Forget did not reset current to 0")
	}
}

func TestForget_DisablesUnreadRune(t *testing.T) {
	rd := newReaderFromString("hello")
	rd.ReadRune()
	rd.Forget()

	if err := rd.UnreadRune(); err != ErrInvalidUnreadRune {
		t.Errorf("expected ErrInvalidUnreadRune after Forget, got %v", err)
	}
}

func TestForget_ContinuesReading(t *testing.T) {
	rd := newReaderFromString("hello")
	rd.ReadRune() // 'h'
	rd.ReadRune() // 'e'
	rd.Forget()

	r, _, err := rd.ReadRune()
	if err != nil {
		t.Fatalf("unexpected error after Forget: %v", err)
	}
	if r != 'l' {
		t.Errorf("got %q after Forget, want 'l'", r)
	}
}

func TestForget_OnEmptyBuffer(t *testing.T) {
	rd := newReaderFromString("hi")
	// Should not panic on empty/no-op Forget
	rd.Forget()

	r, _, err := rd.ReadRune()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r != 'h' {
		t.Errorf("got %q, want 'h'", r)
	}
}

// ---------------------------------------------------------------------------
// Combined / integration scenarios
// ---------------------------------------------------------------------------

func TestReadAfterUnread_FullSequence(t *testing.T) {
	rd := newReaderFromString("abc")

	r1, _, _ := rd.ReadRune() // 'a'
	r2, _, _ := rd.ReadRune() // 'b'

	rd.UnreadRune() // back to 'b'
	rd.UnreadRune() // back to 'a'

	r3, _, _ := rd.ReadRune() // 'a' again
	r4, _, _ := rd.ReadRune() // 'b' again
	r5, _, _ := rd.ReadRune() // 'c'

	if r1 != r3 || r2 != r4 || r5 != 'c' {
		t.Errorf("sequence mismatch: %q %q %q %q %q", r1, r2, r3, r4, r5)
	}
}

func TestPeekThenRead(t *testing.T) {
	rd := newReaderFromString("hello")

	peeked, _ := rd.Peek(3)
	p := make([]rune, 5)
	rd.Read(p)

	if string(peeked) != "hel" {
		t.Errorf("Peek returned %q, want %q", string(peeked), "hel")
	}
	if string(p) != "hello" {
		t.Errorf("Read returned %q, want %q", string(p), "hello")
	}
}

func TestDiscardThenRead(t *testing.T) {
	rd := newReaderFromString("hello world")
	rd.Discard(6) // skip "hello "

	p := make([]rune, 5)
	rd.Read(p)

	if string(p) != "world" {
		t.Errorf("got %q after Discard, want %q", string(p), "world")
	}
}

func TestForgetThenPeek(t *testing.T) {
	rd := newReaderFromString("hello")
	rd.ReadRune() // 'h'
	rd.Forget()

	runes, err := rd.Peek(2)
	if err != nil {
		t.Fatalf("unexpected error after Forget+Peek: %v", err)
	}
	if string(runes) != "el" {
		t.Errorf("got %q after Forget+Peek, want %q", string(runes), "el")
	}
}

func TestMixedMultibyteAndASCII(t *testing.T) {
	// Mix of ASCII and multi-byte runes
	input := "a日b本c"
	rd := newReaderFromString(input)
	p := make([]rune, utf8.RuneCountInString(input))
	rd.Read(p)

	if string(p) != input {
		t.Errorf("got %q, want %q", string(p), input)
	}
}
