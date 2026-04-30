# Vip

A simple scripting lang

```
import { println } from "std/console";
import dom from "std/dom";

fn main(){
   let container = dom.getElementById("#container");
   if container != null {
    container.textContent = "Hello, World";
   }

  let btn = dom.querySelector("button");
  if btn != null {
    btn.addEventHandler("click",(ev dom.ClickEvent)=>{
        println("Hello, there");
    });
  }
}

```
# Spec

Imports

```
import { namedExport } from "fileimport";

import defaultExport from "fileimport";
```

### Functions 

```
fn namedFunc(){}

fn namedFunc(): int {
    return 1;
}

fn namedFunc(value string, value string) {}

fn namedFunc(): (int, int) {
    return 1, 2;
}

let annonFn = () => {}
```

#### Structs 

```
struct Example { 
    name: string;
    prop: stirng[];
}

impl Example {
    fn hello(self, arg string){

    }
}

```

### Variables

```
let = 1;
let mut = 2;
```

### Conditional

```
if true {

} else if false {

} else {

}

true ? "A" : "B"
```

### loops 

```
    loop {

    }

    for x in [] {}

    while x < 0 {}
```

### match 


```
match x {
    'a' => {}
    _ => {}
}
```

### Enum 

```
enum Example {
    A,
    B,
    C
}
```

### types 

```
uint
int
bool
u64
u32
u8
i32
i64
i8
char
string
```

arrays

```
let a = ["a"];
a.len();

let a: string[] = [];
let a string[4] = [];
```