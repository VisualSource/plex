package typeschecker

import (
	"errors"
	"fmt"

	"github.com/VisualSource/plex/internal/script"
)

var (
	ErrUnknownTypename = errors.New("unknown typename")
)

type TypeError struct {
	EndLocation, StartLocation script.Position
	Reason                     error
}

func (r *TypeError) Error() string {
	return fmt.Sprintf("%d:%d-%d:%d | %s", r.StartLocation.Col, r.StartLocation.Row, r.EndLocation.Row, r.EndLocation.Col, r.Reason)
}

func NewTypeError(node script.AstNode, reason error) *TypeError {
	start, end := node.Range()

	return &TypeError{
		StartLocation: start,
		EndLocation:   end,
		Reason:        reason,
	}
}
