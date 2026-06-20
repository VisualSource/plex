# Tree-walking interpreter implemented; WASM compiler stub started

User implemented the full interpreter in a separate `script_interpreter` package with proper separation of concerns (values.go, env.go, eval.go). Test evaluates multiplication via repeated addition with a while loop — demonstrates fn, while, variable declarations, assignment, and return all working together.

Also started a WASM compiler stub (`compiler/wasm.go`) targeting wazero — unprompted, shows the user is thinking ahead to compilation.

**Bugs present:**
- FunctionCall uses NewEnvironment(env) (call-site) instead of fn.Env — breaks closures
- WhileStatement catches returnSignal — return inside a loop escapes to the loop instead of the function
- fmt.Errorf uses %l (invalid) instead of %T

**Implications:** Core interpreter mechanics are understood. Ready for Lesson 5: structs + native functions. The WASM stub shows the user is ready to start thinking about code generation; that should be Lesson 6.
