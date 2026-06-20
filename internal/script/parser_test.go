package script_test

import (
	"strings"
	"testing"

	"github.com/VisualSource/plex/internal/script"
)

func TestParser_Parse(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{
			name:  "operator precedence",
			input: "1 + 2 * 4;",
			want: `
				Program
					Stmts
						BinaryExpression
							Left
								NumberLiteral(1)
							Operator(+)
							Right
								BinaryExpression
									Left
										NumberLiteral(2)
									Operator(*)
									Right
										NumberLiteral(4)
			`,
		},
		{
			name:  "function call",
			input: "someFunction(1,2);",
			want: `
				Program
					Stmts
						FunctionCall
							Callee
								Identifier(someFunction)
							Args
								NumberLiteral(1)
								NumberLiteral(2)
			`,
		},
		{
			name:  "method access",
			input: "someObject.helloWorld;",
			want: `
				Program
					Stmts
						MemberAccess
							Object
								Identifier(someObject)
							Field(helloWorld)
			`,
		},
		{
			name:  "function def: empty",
			input: `fn hello(){}`,
			want: `
				Program
					Stmts
						FunctionDeclaration
							Name(hello)
							Params
							ReturnType
							Body
								Block
									Stmts
			`,
		},
		{
			name:  "function def: single",
			input: `fn hello(param: string){}`,
			want: `
				Program
					Stmts
						FunctionDeclaration
							Name(hello)
							Params
								Parameter
									Name(param)
									Type
										Type
											Name(string)
							ReturnType
							Body
								Block
									Stmts
			`,
		},
		{
			name:  "function def",
			input: `fn hello(param: string, other: int){}`,
			want: `
				Program
					Stmts
						FunctionDeclaration
							Name(hello)
							Params
								Parameter
									Name(param)
									Type
										Type
											Name(string)
								Parameter
									Name(other)
									Type
										Type
											Name(int)
							ReturnType
							Body
								Block
									Stmts
			`,
		},
		{
			name:  "struct def: no fields",
			input: "struct Hello { }",
			want: `
				Program
					Stmts
						StructStatement
							Name(Hello)
							Fields
			`,
		},
		{
			name: "struct def: with fields",
			input: `
				struct Hello {
					field: string;
					field2: string;
				}
			`,
			want: `
				Program
					Stmts
						StructStatement
							Name(Hello)
							Fields
								Parameter
									Name(field)
									Type
										Type
											Name(string)
								Parameter
									Name(field2)
									Type
										Type
											Name(string)
			`,
		},
		{
			name:  "struct impl: empty",
			input: `impl Hello {}`,
			want: `
				Program
					Stmts
						StructImplStatement
							Name(Hello)
							Methods
			`,
		},
		{
			name: "struct impl: single",
			input: `
				impl Hello {
					fn say(){}
				}
			`,
			want: `
				Program
					Stmts
						StructImplStatement
							Name(Hello)
							Methods
								FunctionDeclaration
									Name(say)
									Params
									ReturnType
									Body
										Block
											Stmts
			`,
		},
		{
			name: "struct impl: multi",
			input: `
				impl Hello {
					fn say(){}
					fn act(){}
				}
			`,
			want: `
				Program
					Stmts
						StructImplStatement
							Name(Hello)
							Methods
								FunctionDeclaration
									Name(say)
									Params
									ReturnType
									Body
										Block
											Stmts
								FunctionDeclaration
									Name(act)
									Params
									ReturnType
									Body
										Block
											Stmts
			`,
		},
		{
			name:  "if signle",
			input: `if true {}`,
			want: `
				Program
					Stmts
						IfStatement
							Condition
								BooleanLiteral
									Value(true)
							Body
								Block
									Stmts
							Else
			`,
		},
		{
			name:  "if with else",
			input: `if true {} else {}`,
			want: `
				Program
					Stmts
						IfStatement
							Condition
								BooleanLiteral
									Value(true)
							Body
								Block
									Stmts
							Else
								Block
									Stmts
			`,
		},
		{
			name:  "if with else if",
			input: `if true {} else if false {} else {}`,
			want: `
				Program
					Stmts
						IfStatement
							Condition
								BooleanLiteral
									Value(true)
							Body
								Block
									Stmts
							Else
								IfStatement
									Condition
										BooleanLiteral
									Body
										Block
											Stmts
									Else
										Block
											Stmts
			`,
		},
		{
			name:  "Indexing",
			input: `target[index];`,
			want: `
				Program
					Stmts
						ArrayAccess
							Target
								Identifier(target)
							Index
								Identifier(index)
			`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := script.NewParser(getTokens(tt.input))
			got, gotErr := p.Parse()
			if gotErr != nil {
				if !tt.wantErr {
					t.Errorf("Parse() failed: %v", gotErr)
				}
				return
			}

			if tt.wantErr {
				t.Fatal("Parse() succeeded unexpectedly")
			}
			if diff, nw, ng := diffAst(got, tt.want); diff != "" {
				t.Errorf("Parse() mismatch (-want +got):\n%s\n\nWant\n%s\n\nGot:\n%s", diff, nw, ng)
			}
		})
	}
}

func getTokens(input string) []script.Token {
	tok := script.NewTokenizer(strings.NewReader(input))

	tokens, err := tok.Tokenize()
	if err != nil {
		panic(err)
	}

	return tokens
}
