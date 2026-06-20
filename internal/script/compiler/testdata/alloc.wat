(module
 (func $alloc (param $size i32) (result i32)
  (local $ptr i32)
  global.get $heapPtr
  local.set $ptr
  global.get $heapPtr
  local.get $size
  i32.add
  global.set $heapPtr
  local.get $ptr
  return
 )
 (export "alloc" (func $alloc))
)
