package script_test

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/google/go-cmp/cmp"

	"github.com/VisualSource/plex/internal/script"
)

var (
	astNodeType   = reflect.TypeOf((*script.AstNode)(nil)).Elem()
	tokenTypeType = reflect.TypeOf(script.TokenType(0))
)

var tokenSymbol = map[script.TokenType]string{
	script.TokenType_Plus:               "+",
	script.TokenType_Minus:              "-",
	script.TokenType_Star:               "*",
	script.TokenType_Div:                "/",
	script.TokenType_Mod:                "%",
	script.TokenType_LessThen:           "<",
	script.TokenType_GreaterThen:        ">",
	script.TokenType_LessThenOrEqual:    "<=",
	script.TokenType_GreaterThenOrEqaul: ">=",
	script.TokenType_EqualEqual:         "==",
	script.TokenType_NotEqual:           "!=",
	script.TokenType_AND:                "&&",
	script.TokenType_OR:                 "||",
	script.TokenType_Incrment:           "++",
	script.TokenType_Decrement:          "--",
	script.TokenType_Equal:              "=",
}

func formatAst(node script.AstNode) string {
	var b strings.Builder
	writeNode(&b, reflect.ValueOf(node), 0)
	return b.String()
}

func writeNode(b *strings.Builder, v reflect.Value, depth int) {
	if v.Kind() == reflect.Interface {
		if v.IsNil() {
			return
		}
		v = v.Elem()
	}
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return
		}
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		panic(fmt.Sprintf("formatAst: expected struct, got %s", v.Kind()))
	}

	typeName := v.Type().Name()

	type field struct {
		name string
		val  reflect.Value
	}
	var fields []field
	for _, sf := range reflect.VisibleFields(v.Type()) {
		if sf.Name == "Start" || sf.Name == "End" {
			continue
		}
		fields = append(fields, field{sf.Name, v.FieldByIndex(sf.Index)})
	}

	if len(fields) == 1 && fields[0].val.Kind() == reflect.String {
		writeIndent(b, depth)
		b.WriteString(typeName)
		b.WriteByte('(')
		b.WriteString(fields[0].val.String())
		b.WriteString(")\n")
		return
	}

	writeIndent(b, depth)
	b.WriteString(typeName)
	b.WriteByte('\n')

	for _, f := range fields {
		writeField(b, f.name, f.val, depth+1)
	}
}

func writeField(b *strings.Builder, name string, fv reflect.Value, depth int) {
	switch {
	case fv.Type() == astNodeType:
		writeIndent(b, depth)
		b.WriteString(name)
		b.WriteByte('\n')
		if !fv.IsNil() {
			writeNode(b, fv, depth+1)
		}
	case fv.Kind() == reflect.Slice && fv.Type().Elem() == astNodeType:
		writeIndent(b, depth)
		b.WriteString(name)
		b.WriteByte('\n')
		for i := 0; i < fv.Len(); i++ {
			writeNode(b, fv.Index(i), depth+1)
		}
	case fv.Kind() == reflect.Slice && fv.Type().Elem().Kind() == reflect.String:
		writeIndent(b, depth)
		b.WriteString(name)
		b.WriteByte('\n')
		for i := 0; i < fv.Len(); i++ {
			writeIndent(b, depth+1)
			b.WriteString("Import(")
			b.WriteString(fv.Index(i).String())
			b.WriteString(")\n")
		}
	case fv.Kind() == reflect.String:
		writeIndent(b, depth)
		b.WriteString(name)
		b.WriteByte('(')
		b.WriteString(fv.String())
		b.WriteString(")\n")
	case fv.Kind() == reflect.Bool:
		if fv.Bool() {
			writeIndent(b, depth)
			b.WriteString(name)
			b.WriteString("(true)\n")
		}
	case fv.Type() == tokenTypeType:
		tt := script.TokenType(fv.Uint())
		sym, ok := tokenSymbol[tt]
		if !ok {
			panic(fmt.Sprintf("formatAst: no symbol for TokenType %d", tt))
		}
		writeIndent(b, depth)
		b.WriteString(name)
		b.WriteByte('(')
		b.WriteString(sym)
		b.WriteString(")\n")
	default:
		panic(fmt.Sprintf("formatAst: unhandled field %s (kind=%s type=%s)", name, fv.Kind(), fv.Type()))
	}
}

func writeIndent(b *strings.Builder, depth int) {
	for i := 0; i < depth; i++ {
		b.WriteByte('\t')
	}
}

// normalize strips the longest common leading-whitespace prefix shared by all
// non-blank lines and drops blank lines. The renderer emits canonical
// (tab-indented, no leading prefix) output; want-strings in tests are usually
// embedded in indented Go source, so they share an outer whitespace prefix
// that this trims away.
func normalize(s string) string {
	var lines []string
	for _, l := range strings.Split(s, "\n") {
		if strings.TrimSpace(l) == "" {
			continue
		}
		lines = append(lines, strings.TrimRight(l, " \t"))
	}
	if len(lines) == 0 {
		return ""
	}

	prefix := leadingWhitespace(lines[0])
	for _, l := range lines[1:] {
		prefix = commonPrefix(prefix, leadingWhitespace(l))
		if prefix == "" {
			break
		}
	}

	var b strings.Builder
	for _, l := range lines {
		b.WriteString(strings.TrimPrefix(l, prefix))
		b.WriteByte('\n')
	}
	return b.String()
}

func leadingWhitespace(s string) string {
	for i := 0; i < len(s); i++ {
		if s[i] != ' ' && s[i] != '\t' {
			return s[:i]
		}
	}
	return s
}

func commonPrefix(a, b string) string {
	n := len(a)
	if len(b) < n {
		n = len(b)
	}
	for i := 0; i < n; i++ {
		if a[i] != b[i] {
			return a[:i]
		}
	}
	return a[:n]
}

// diffAst returns "" if got matches want after normalisation; otherwise it
// returns a unified diff suitable for t.Errorf. Want appears as the baseline
// (lines prefixed with "-"); got as the change ("+").
func diffAst(got script.AstNode, want string) string {
	g := normalize(formatAst(got))
	w := normalize(want)
	if g == w {
		return ""
	}
	return cmp.Diff(w, g)
}
