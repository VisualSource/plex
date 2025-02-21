package plex_css

var CSS_COLOR_KEYWORDS = map[string]CssColor{
	"aliceblue":            {R: 240, G: 248, B: 255, A: 255},
	"antiquewhite":         {R: 250, G: 235, B: 215, A: 255},
	"aqua":                 {R: 0, G: 255, B: 255, A: 255},
	"aquamarine":           {R: 127, G: 255, B: 212, A: 255},
	"azure":                {R: 240, G: 255, B: 255, A: 255},
	"beige":                {R: 245, G: 245, B: 220, A: 255},
	"bisque":               {R: 255, G: 228, B: 196, A: 255},
	"black":                {R: 0, G: 0, B: 0, A: 255},
	"blanchedalmond":       {R: 255, G: 235, B: 205, A: 255},
	"blue":                 {R: 0, G: 0, B: 255, A: 255},
	"blueviolet":           {R: 138, G: 43, B: 226, A: 255},
	"brown":                {R: 165, G: 42, B: 42, A: 255},
	"burlywood":            {R: 222, G: 184, B: 135, A: 255},
	"cadetblue":            {R: 95, G: 158, B: 160, A: 255},
	"chartreuse":           {R: 127, G: 255, B: 0, A: 255},
	"chocolate":            {R: 210, G: 105, B: 30, A: 255},
	"coral":                {R: 255, G: 127, B: 80, A: 255},
	"cornflowerblue":       {R: 100, G: 149, B: 237, A: 255},
	"cornsilk":             {R: 255, G: 248, B: 220, A: 255},
	"crimson":              {R: 220, G: 20, B: 60, A: 255},
	"cyan":                 {R: 0, G: 255, B: 255, A: 255},
	"darkblue":             {R: 0, G: 0, B: 139, A: 255},
	"darkcyan":             {R: 0, G: 139, B: 139, A: 255},
	"darkgoldenrod":        {R: 184, G: 134, B: 11, A: 255},
	"darkgray":             {R: 169, G: 169, B: 169, A: 255},
	"darkgreen":            {R: 0, G: 100, B: 0, A: 255},
	"darkgrey":             {R: 169, G: 169, B: 169, A: 255},
	"darkkhaki":            {R: 189, G: 183, B: 107, A: 255},
	"darkmagenta":          {R: 139, G: 0, B: 139, A: 255},
	"darkolivegreen":       {R: 85, G: 107, B: 47, A: 255},
	"darkorange":           {R: 255, G: 140, B: 0, A: 255},
	"darkorchid":           {R: 153, G: 50, B: 204, A: 255},
	"darkred":              {R: 139, G: 0, B: 0, A: 255},
	"darksalmon":           {R: 233, G: 150, B: 122, A: 255},
	"darkseagreen":         {R: 143, G: 188, B: 143, A: 255},
	"darkslateblue":        {R: 72, G: 61, B: 139, A: 255},
	"darkslategray":        {R: 47, G: 79, B: 79, A: 255},
	"darkslategrey":        {R: 47, G: 79, B: 79, A: 255},
	"darkturquoise":        {R: 0, G: 206, B: 209, A: 255},
	"darkviolet":           {R: 148, G: 0, B: 211, A: 255},
	"deeppink":             {R: 255, G: 20, B: 147, A: 255},
	"deepskyblue":          {R: 0, G: 191, B: 255, A: 255},
	"dimgray":              {R: 105, G: 105, B: 105, A: 255},
	"dimgrey":              {R: 105, G: 105, B: 105, A: 255},
	"dodgerblue":           {R: 30, G: 144, B: 255, A: 255},
	"firebrick":            {R: 178, G: 34, B: 34, A: 255},
	"floralwhite":          {R: 255, G: 250, B: 240, A: 255},
	"forestgreen":          {R: 34, G: 139, B: 34, A: 255},
	"fuchsia":              {R: 255, G: 0, B: 255, A: 255},
	"gainsboro":            {R: 220, G: 220, B: 220, A: 255},
	"ghostwhite":           {R: 248, G: 248, B: 255, A: 255},
	"gold":                 {R: 255, G: 215, B: 0, A: 255},
	"goldenrod":            {R: 218, G: 165, B: 32, A: 255},
	"gray":                 {R: 128, G: 128, B: 128, A: 255},
	"green":                {R: 0, G: 128, B: 0, A: 255},
	"greenyellow":          {R: 173, G: 255, B: 47, A: 255},
	"grey":                 {R: 128, G: 128, B: 128, A: 255},
	"honeydew":             {R: 240, G: 255, B: 240, A: 255},
	"hotpink":              {R: 255, G: 105, B: 180, A: 255},
	"indianred":            {R: 205, G: 92, B: 92, A: 255},
	"indigo":               {R: 75, G: 0, B: 130, A: 255},
	"ivory":                {R: 255, G: 255, B: 240, A: 255},
	"khaki":                {R: 240, G: 230, B: 140, A: 255},
	"lavender":             {R: 230, G: 230, B: 250, A: 255},
	"lavenderblush":        {R: 255, G: 240, B: 245, A: 255},
	"lawngreen":            {R: 124, G: 252, B: 0, A: 255},
	"lemonchiffon":         {R: 255, G: 250, B: 205, A: 255},
	"lightblue":            {R: 173, G: 216, B: 230, A: 255},
	"lightcoral":           {R: 240, G: 128, B: 128, A: 255},
	"lightcyan":            {R: 224, G: 255, B: 255, A: 255},
	"lightgoldenrodyellow": {R: 250, G: 250, B: 210, A: 255},
	"lightgray":            {R: 211, G: 211, B: 211, A: 255},
	"lightgreen":           {R: 144, G: 238, B: 144, A: 255},
	"lightgrey":            {R: 211, G: 211, B: 211, A: 255},
	"lightpink":            {R: 255, G: 182, B: 193, A: 255},
	"lightsalmon":          {R: 255, G: 160, B: 122, A: 255},
	"lightseagreen":        {R: 32, G: 178, B: 170, A: 255},
	"lightskyblue":         {R: 135, G: 206, B: 250, A: 255},
	"lightslategray":       {R: 119, G: 136, B: 153, A: 255},
	"lightslategrey":       {R: 119, G: 136, B: 153, A: 255},
	"lightsteelblue":       {R: 176, G: 196, B: 222, A: 255},
	"lightyellow":          {R: 255, G: 255, B: 224, A: 255},
	"lime":                 {R: 0, G: 255, B: 0, A: 255},
	"limegreen":            {R: 50, G: 205, B: 50, A: 255},
	"linen":                {R: 250, G: 240, B: 230, A: 255},
	"magenta":              {R: 255, G: 0, B: 255, A: 255},
	"maroon":               {R: 128, G: 0, B: 0, A: 255},
	"mediumaquamarine":     {R: 102, G: 205, B: 170, A: 255},
	"mediumblue":           {R: 0, G: 0, B: 205, A: 255},
	"mediumorchid":         {R: 186, G: 85, B: 211, A: 255},
	"mediumpurple":         {R: 147, G: 112, B: 219, A: 255},
	"mediumseagreen":       {R: 60, G: 179, B: 113, A: 255},
	"mediumslateblue":      {R: 123, G: 104, B: 238, A: 255},
	"mediumspringgreen":    {R: 0, G: 250, B: 154, A: 255},
	"mediumturquoise":      {R: 72, G: 209, B: 204, A: 255},
	"mediumvioletred":      {R: 199, G: 21, B: 133, A: 255},
	"midnightblue":         {R: 25, G: 25, B: 112, A: 255},
	"mintcream":            {R: 245, G: 255, B: 250, A: 255},
	"mistyrose":            {R: 255, G: 228, B: 225, A: 255},
	"moccasin":             {R: 255, G: 228, B: 181, A: 255},
	"navajowhite":          {R: 255, G: 222, B: 173, A: 255},
	"navy":                 {R: 0, G: 0, B: 128, A: 255},
	"oldlace":              {R: 253, G: 245, B: 230, A: 255},
	"olive":                {R: 128, G: 128, B: 0, A: 255},
	"olivedrab":            {R: 107, G: 142, B: 35, A: 255},
	"orange":               {R: 255, G: 165, B: 0, A: 255},
	"orangered":            {R: 255, G: 69, B: 0, A: 255},
	"orchid":               {R: 218, G: 112, B: 214, A: 255},
	"palegoldenrod":        {R: 238, G: 232, B: 170, A: 255},
	"palegreen":            {R: 152, G: 251, B: 152, A: 255},
	"paleturquoise":        {R: 175, G: 238, B: 238, A: 255},
	"palevioletred":        {R: 219, G: 112, B: 147, A: 255},
	"papayawhip":           {R: 255, G: 239, B: 213, A: 255},
	"peachpuff":            {R: 255, G: 218, B: 185, A: 255},
	"peru":                 {R: 205, G: 133, B: 63, A: 255},
	"pink":                 {R: 255, G: 192, B: 203, A: 255},
	"plum":                 {R: 221, G: 160, B: 221, A: 255},
	"powderblue":           {R: 176, G: 224, B: 230, A: 255},
	"purple":               {R: 128, G: 0, B: 128, A: 255},
	"rebeccapurple":        {R: 102, G: 51, B: 153, A: 255},
	"red":                  {R: 255, G: 0, B: 0, A: 255},
	"rosybrown":            {R: 188, G: 143, B: 143, A: 255},
	"royalblue":            {R: 65, G: 105, B: 225, A: 255},
	"saddlebrown":          {R: 139, G: 69, B: 19, A: 255},
	"salmon":               {R: 250, G: 128, B: 114, A: 255},
	"sandybrown":           {R: 244, G: 164, B: 96, A: 255},
	"seagreen":             {R: 46, G: 139, B: 87, A: 255},
	"seashell":             {R: 255, G: 245, B: 238, A: 255},
	"sienna":               {R: 160, G: 82, B: 45, A: 255},
	"silver":               {R: 192, G: 192, B: 192, A: 255},
	"skyblue":              {R: 135, G: 206, B: 235, A: 255},
	"slateblue":            {R: 106, G: 90, B: 205, A: 255},
	"slategray":            {R: 112, G: 128, B: 144, A: 255},
	"slategrey":            {R: 112, G: 128, B: 144, A: 255},
	"snow":                 {R: 255, G: 250, B: 250, A: 255},
	"springgreen":          {R: 0, G: 255, B: 127, A: 255},
	"steelblue":            {R: 70, G: 130, B: 180, A: 255},
	"tan":                  {R: 210, G: 180, B: 140, A: 255},
	"teal":                 {R: 0, G: 128, B: 128, A: 255},
	"thistle":              {R: 216, G: 191, B: 216, A: 255},
	"tomato":               {R: 255, G: 99, B: 71, A: 255},
	"turquoise":            {R: 64, G: 224, B: 208, A: 255},
	"violet":               {R: 238, G: 130, B: 238, A: 255},
	"wheat":                {R: 245, G: 222, B: 179, A: 255},
	"white":                {R: 255, G: 255, B: 255, A: 255},
	"whitesmoke":           {R: 245, G: 245, B: 245, A: 255},
	"yellow":               {R: 255, G: 255, B: 0, A: 255},
	"yellowgreen":          {R: 154, G: 205, B: 50, A: 255},
}

func f(n, hue, light, sat int) int {
	k := (n + hue/30) % 12
	a := sat * min(light, 1-light)
	return light - a*max(-1, min(k-3, 9-k, 1))
}

/*
Converting an HSL color to sRGB is straightforward mathematically.

It returns an array of three numbers representing the red, green, and blue channels of the colors,
normalized to the range [0, 1].

This code assumes that parse-time clamping of negative saturation has already been applied.

Source:  https://www.w3.org/TR/css-color-4/#hsl-to-rgb
*/
func HslToRgb(hue, sat, light int) (int, int, int) {
	hue = hue % 360

	if hue < 0 {
		hue += 360
	}

	sat /= 100
	light /= 100

	return f(0, hue, light, sat), f(8, hue, light, sat), f(4, hue, light, sat)
}
