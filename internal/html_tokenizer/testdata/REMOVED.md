### unicodeCharsProblematic

```
{"description": "Invalid Unicode character U+DFFF",
"doubleEscaped":true,
"input": "\\uDFFF",
"output":[["Character", "\\uDFFF"]],
"errors":[
    { "code": "surrogate-in-input-stream", "line": 1, "col": 1 }
]},

{"description": "Invalid Unicode character U+D800",
"doubleEscaped":true,
"input": "\\uD800",
"output":[["Character", "\\uD800"]],
"errors":[
    { "code": "surrogate-in-input-stream", "line": 1, "col": 1 }
]},

{"description": "Invalid Unicode character U+DFFF with valid preceding character",
"doubleEscaped":true,
"input": "a\\uDFFF",
"output":[["Character", "a\\uDFFF"]],
"errors":[
    { "code": "surrogate-in-input-stream", "line": 1, "col": 2 }
]},

{"description": "Invalid Unicode character U+D800 with valid following character",
"doubleEscaped":true,
"input": "\\uD800a",
"output":[["Character", "\\uD800a"]],
"errors":[
    { "code": "surrogate-in-input-stream", "line": 1, "col": 1 }
]},
```