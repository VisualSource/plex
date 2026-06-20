(module
 (memory 1)
 (export "memory" (memory 0))
 (global $heapPtr (mut i32) (i32.const 0))
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
 (func $main
  (local $a i32)
  (local $__array_tmp0 i32)
  (local $b i64)
  ;; array literal
  i32.const 20
  call $alloc
  local.tee $__array_tmp0
  i32.const 2
  i32.store
  ;; insert element 0
  local.get $__array_tmp0
  i32.const 4
  i32.add
  i64.const 1
  i64.store
  ;; insert end
  ;; insert element 1
  local.get $__array_tmp0
  i32.const 12
  i32.add
  i64.const 2
  i64.store
  ;; insert end
  local.get $__array_tmp0
  ;; end array literal
  local.set $a
  i64.const 1
  local.set $b
  ;; array access
  local.get $a
  i32.const 4
  i32.add
  local.get $b
  i32.wrap_i64
  i32.const 8
  i32.mul
  i32.add
  i64.load
  ;; end array access
  drop
 )
 (export "main" (func $main))
)
