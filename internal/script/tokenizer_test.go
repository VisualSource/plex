package script_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/VisualSource/plex/internal/script"
)

func Test_Tokenizer(t *testing.T) {
	comments := ` 
	1.2e3 1 1.11 .11 +111 -111 1e2 1E2 +1e2 -1e2 .1e2 -.2e3 +.2e4 
	import from struct if else while let mut fn
	null float int uint bool
	{ } [ ] ( ) . : ; ?
	|| && != == = % < > >= <= => - + * /
	"string" ident
	// comment
	/* */ 
	/*
	  Multiline comment
	
	  d
	*/
	`
	tokenizer := script.NewTokenizer(strings.NewReader(comments))

	tokens, err := tokenizer.Tokenize()
	if err != nil {
		t.Fatalf("%v", err)
	}

	bytes, err := json.MarshalIndent(tokens, "", "")
	if err != nil {
		panic(err)
	}

	t.Logf("%s", string(bytes))
}
