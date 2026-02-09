import characterReference from "../resources/character_reference.json" with { type: "json" }

const refs = new Map<string,{ codepoints: number[], characters: string;}>();

for (const [key,value] of Object.entries(characterReference)){
    const stripedKey = key.replace(";","").replace("&","");

    if(!refs.has(stripedKey)){
        refs.set(stripedKey,value);
    }
}

const templete = `case "{REF_NAME}":
                        return {CODE_POINT}`;


const items = Object.entries(refs).map((([key,value])=>( templete.replace("{REF_NAME}",key).replace("{CODE_POINT}",value.codepoints.toString()) ))).join("\n")

console.log(refs);