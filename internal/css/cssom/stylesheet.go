package cssom

import "github.com/VisualSource/plex/internal/utils"

type Stylesheet struct {
	Location utils.StringOption
	Parent   *Stylesheet
	Value    []Rule
}

func NewStylesheet(location utils.StringOption) *Stylesheet {
	return &Stylesheet{
		Location: location,
	}
}

type Rule struct{}

type Declaration struct {
	Important bool
	Name      string
	Value     []Component
}

func NewDeclaration(name string) *Declaration {
	return &Declaration{
		Name: name,
	}
}

type Component struct{}
