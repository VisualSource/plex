# WASM Binary Skeleton + LEB128

User completed lesson 24. They now understand:

- A `.wasm` file is an 8-byte header (`00 61 73 6D 01 00 00 00`) followed by zero or more length-prefixed sections in a fixed numeric order (Type=1, Function=3, Memory=5, Global=6, Export=7, Code=10, Data=11).
- Fixed-width integers in the binary format are little-endian (the version field is `01 00 00 00`, not `00 00 00 01`).
- LEB128 is the variable-width encoding used for all counts, lengths, and indices in the binary format. Unsigned variant: 7-bit groups, low-group-first, continuation bit on every non-final byte. Signed variant: same shape, but the high bit of the final group must equal the value's sign so the decoder can sign-extend correctly.

User has a parallel `internal/script/compiler/binary` package that emits a valid (empty) `.wasm` module via the `-target wasm-bin` CLI flag, alongside the existing WAT backend. Confirmed by byte-diff against `wat2wasm` output for `(module)`.

**Implications:** User has the two foundational tools every later section will rely on — the section wrapper pattern (id + LEB128 size + body) and LEB128 encoding. They're ready to encode actual content. Next: function signatures (Type section), then function bodies (Function + Code), then exports — each lesson adds one section between `version` and EOF.
