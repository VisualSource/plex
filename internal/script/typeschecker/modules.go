package typeschecker

import (
	"errors"
	"fmt"

	"github.com/VisualSource/plex/internal/script"
)

func importModule(c *Checker, node *script.ImportStatement) {
	seen := make(map[string]bool)
	switch node.Source {
	case "plex:console":
		for _, imp := range node.Imports {
			switch imp {
			case "print":
				info := newFuncInfo(c.scope)

				info.ReturnType = &script.Type{
					Kind: script.TypeKind_Void,
				}

				info.Args["arg"] = &orderedItem{
					Pos: 0,
					Type: &script.Type{
						Kind: script.TypeKind_String,
					},
				}

				c.funcs[imp] = info
			}
		}
	case "plex:globals":

		for _, imp := range node.Imports {
			switch imp {
			case "heapPtr":
				if _, ok := seen[imp]; ok {
					c.error(node, fmt.Errorf("already imported %s", imp))
					continue
				}
				seen[imp] = true
				c.scope.Set("heapPtr", &script.Type{
					Kind: script.TypeKind_I32,
				})
			default:
				c.error(node, fmt.Errorf("unknown import '%s'", imp))

			}
		}
	default:
		c.error(node, errors.New("failed to import source file"))
	}
}
