# argile

<keyword> import|from|struct|if|else|while|let|mut|fn|null

<structDefinition> = <keyword:export>? <keyword:struct> <ident> <scope>
    <ident> <token:':'> <type> <token:';'>*
</scope>

<condition> = <expression> (<token:'||'>|<token:'&&'> <expression>)*

<ifStatement> = <keyword:if> <condition> </scope> <keyword:else> (</scope>) | (<ifStatement>) 

<ternaryStatement> = <expr> <token:'?'> <expr> <token:':'> <expr>

<literal> = <string>|<number>|<bool>

<functionParams> = <token:'('> (<ident> <token:':'> <type> (<token:'='> <literal>)? (<token:','> <ident> <token:':'> <type>)*  )* <token:')'>
<functionDefinition> = <keyword:export>? <keyword:fn> <ident> <functionParams>  (<token:'->'> <type>)? </scope>
<arrowFunction> = <functionParams> (<token:':' <type>)? <token:'=>'> (</scope> | <expression> )

<whileStatement> = <keyword:while> <condition> </scope>

<op> = <expression> (<token:'/'>|<token:'*'>|<token:'+'>|<token:"-">|<token:'%'>|<token:'>'>|<token;'<'>) <expression>

<expression> = <literal> | <op>

<variableDefinition> = <keyword:let> <keyword:mut>? (<token:':'> <type>)? = <expression> <token:';'>

<functionCall> = <ident> <token:'('> <expression>* <token:')'>
<importStatement> = <keyword:import> <token:'{'> (<ident> (<token:','> <ident>)* )? <token:'}'> <keyword:from> <string> <token:';'>


let <ident> = 1;
let <ident>: <type> = <literal>;
let mut <ident> = 1;

<string> "STRING"
<int|int64|int32|u8|u32|u64> 1
<float|float64|float32> 1.11111
<ident>[] int[]

!= == = || < > >= <= =>