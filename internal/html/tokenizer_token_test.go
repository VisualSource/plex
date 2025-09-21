package html_test

import (
	"testing"

	"github.com/VisualSource/plex/internal/html"
)

func TestToken_DOCTYPE_AppendStringToName(t *testing.T) {
	tests := []struct {
		name string // description of this test case
		// Named input parameters for receiver constructor.
		value string
		// Named input parameters for target function.
		test    func(*html.Token) bool
		wantErr bool
	}{
		{
			name:    "Append value",
			wantErr: false,
			value:   "doc",
			test: func(t *html.Token) bool {
				v, _ := t.DOCTYPE_GetName()
				return *v == "doc"
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			to := html.NewDOCTYPEToken()
			to.DOCTYPE_SetName("")
			gotErr := to.DOCTYPE_AppendStringToName(tt.value)
			if gotErr != nil {
				if !tt.wantErr {
					t.Errorf("DOCTYPE_AppendStringToName() failed: %v", gotErr)
				}
				return
			}
			if tt.wantErr {
				t.Fatal("DOCTYPE_AppendStringToName() succeeded unexpectedly")
			}
			if !tt.test(to) {
				t.Fatal("DOCTYPE_AppendStringToName() failed to have correct value")
			}
		})
	}
}
