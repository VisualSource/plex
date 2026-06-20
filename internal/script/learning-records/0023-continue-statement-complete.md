# continue Statement — WAT Backward Jump to Loop Label

User implemented `continue` in lesson 23. Key insight: in WAT, `br` to a `loop` label is a backward jump (restarts the loop), while `br` to a `block` label is a forward jump (exits it). `continue` targets the loop label (`$loopN`), `break` targets the block label (`$blockN`) — same opcode, opposite direction. Implementation required a `continueStack` parallel to `breakStack` in the Compiler struct, populated by `pushBreakLabel`.

**Implications:** User understands WAT's structured control flow model at the label level. This knowledge directly motivates the body-block pattern needed for `for` loops (lesson 24), where `continue` must land at the increment rather than the loop top.
