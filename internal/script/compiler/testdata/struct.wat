(module
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
 (func $make_point (param $x f64) (param $y f64) (result i32)
  (local $__struct_Point_tmp0 i32)
  ;; start struct(Point) init
  i32.const 16
  call $alloc
  local.tee $__struct_Point_tmp0
  ;; set field x
  i32.const 0
  i32.add
  local.get $x
  f64.store
  ;; end
  ;; set field y
  local.get $__struct_Point_tmp0
  i32.const 8
  i32.add
  local.get $y
  f64.store
  ;; end
  local.get $__struct_Point_tmp0
  ;; end struct(Point) int
  return
 )
 (export "make_point" (func $make_point))
 (func $get_x (param $p i32) (result f64)
  local.get $p
  i32.const 0
  i32.add
  f64.load
  return
 )
 (export "get_x" (func $get_x))
)
