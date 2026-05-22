### CSS parser tests

Tests updated to reflect the 2026 W3C CSS Syntax Module Level 3 editor's draft:
https://drafts.csswg.org/css-syntax/

Original sources (2021 CR):
- https://github.com/CourtBouillon/css-parsing-tests
- https://github.com/servo/rust-cssparser

### Result representation

AST nodes (the results of parsing) are represented in JSON as follow. This representation was chosen to be compact (and thus less annoying to write by hand) while staying unambiguous. For example, the difference between @import and \@import is not lost: they are represented as ["at-keyword", "import"] and ["ident", "@import"], respectively.

#### Rules and declarations

At-rule
    An array of length 4: the string "at-rule", the name (value of the at-keyword) as a string, the prelude as a nested array of component values, and the optional block as a nested array of component value, or null.
Qualified rule
    An array of length 3: the string "qualified rule", the prelude as a nested array of component values, and the block as a nested array of component value.
Declaration
    An array of length 4: the string "declaration", the name as a string, the value as a nested array of component values, and a the important flag as a boolean.

#### Component values

<ident>
    Array of length 2: the string "ident", and the value as a string.
<at-keyword>
    Array of length 2: the string "at-keyword", and the value as a string.
<hash>
    Array of length 3: the string "hash", the value as a string, and the type as the string "id" or "unrestricted".
<string>
    Array of length 2: the string "string", and the value as a string.
<bad-string>
    Array of length 1: the string "bad-string".
<url>
    Array of length 2: the string "url", and the value as a string.
<bad-url>
    Array of length 1: the string "bad-url".
<delim>
    The value as a one-character string.
<number>
    Array of length 4: the string "number", the representation as a string, the value as a number, and the type as the string "integer" or "number".
<percentage>
    Array of length 4: the string "percentage", the representation as a string, the value as a number, and the type as the string "integer" or "number".
<dimension>
    Array of length 4: the string "dimension", the representation as a string, the value as a number, the type as the string "integer" or "number", and the unit as a string.
<include-match>
    The string "~=".
<dash-match>
    The string "|=".
<prefix-match>
    The string "^=".
<suffix-match>
    The string "$=".
<substring-match>
    The string "*=".
<column>
    The string "||".
<whitespace>
    The string " " (a single space.)
<CDO>
    The string "<!--".
<CDC>
    The string "-->".
<colon>
    The string ":".
<semicolon>
    The string ";".
<comma>
    The string ",".
{} block
    An array of length N+1: the string "{}" followed by the N component values of the block’s content.
[] block
    An array of length N+1: the string "[]" followed by the N component values of the block’s content.
() block
    An array of length N+1: the string "()" followed by the N component values of the block’s content.
Function
    An array of length N+2: the string "function" and the name of the function as a string followed by the N component values of the function’s arguments.
<bad-string>
    The array of two strings ["error", "bad-string"].
<bad-url>
    The array of two strings ["error", "bad-url"].
Unmatched <}>
    The array of two strings ["error", "}"].
Unmatched <]>
    The array of two strings ["error", "]"].
Unmatched <)>
    The array of two strings ["error", ")"]. 