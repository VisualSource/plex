(module
  (memory 1)
  (global $heapPtr (mut i32) (i32.const 0))
  ...
  (func $make_point (param $x f64) (param $y f64) (result i32)
    i32.const 16
    call $alloc
    local.tee $tmp
    i32.const 0
    i32.add
    local.get $x
    f64.store
    local.get $tmp
    i32.const 8
    i32.add
    local.get $y
    f64.store
    local.get $tmp
  )
  (func $get_x (param $p i32) (result f64)
    local.get $p
    i32.const 0
    i32.add
    f64.load
  )
)