(module
 (func $getTrue (result i32)
  i32.const 1
  return
 )
 (export "getTrue" (func $getTrue))
 (func $getTrue (result i32)
  i32.const 0
  return
 )
 (export "getTrue" (func $getTrue))
)
