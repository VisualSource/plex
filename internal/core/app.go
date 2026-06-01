package core

import (
	"context"
	"image"
	"image/color"
	"log/slog"
	"time"

	"gioui.org/app"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"github.com/VisualSource/plex/internal/layouts/widgets"
)

var (
	background = color.NRGBA{R: 0xC0, G: 0xC0, B: 0xC0, A: 0xFF}
	red        = color.NRGBA{R: 0xC0, G: 0x40, B: 0x40, A: 0xFF}
	green      = color.NRGBA{R: 0x40, G: 0xC0, B: 0x40, A: 0xFF}
	blue       = color.NRGBA{R: 0x40, G: 0x40, B: 0xC0, A: 0xFF}
)

func ColorBox(gtx layout.Context, size image.Point, color color.NRGBA) layout.Dimensions {
	defer clip.Rect{Max: size}.Push(gtx.Ops).Pop()
	paint.ColorOp{Color: color}.Add(gtx.Ops)
	paint.PaintOp{}.Add(gtx.Ops)
	return layout.Dimensions{Size: size}
}

func stacked(gtx layout.Context) layout.Dimensions {
	return layout.Stack{}.Layout(gtx,
		// Force widget to the same size as the second.
		layout.Expanded(func(gtx layout.Context) layout.Dimensions {
			// This will have a minimum constraint of 100x100.
			return ColorBox(gtx, gtx.Constraints.Min, red)
		}),
		layout.Stacked(func(gtx layout.Context) layout.Dimensions {
			return ColorBox(gtx, image.Pt(100, 30), green)
		}),
		layout.Stacked(func(gtx layout.Context) layout.Dimensions {
			return ColorBox(gtx, image.Pt(30, 100), blue)
		}),
	)
}

func StartPlex(ctx context.Context, logger *slog.Logger, window *app.Window) error {
	startTime := time.Now()
	tree := renderHtml(`
	<!DOCTYPE html>
	<html lang="en">
	<head>
		<meta charset="UTF-8">
		<meta name="viewport" content="width=device-width, initial-scale=1.0">
		<title>Document</title>
	</head>
	<body>
		
	</body>
	</html>
	`, `
	body { height: 100px; width: 100px; background-color: red; }
	head { display: none; }
	`)

	elapsed := time.Since(startTime)

	logger.DebugContext(ctx, "html -> css -> cssom -> styletree -> layout: time", slog.String("time", elapsed.String()))

	var ops op.Ops
	for {
		switch e := window.Event().(type) {
		case app.DestroyEvent:
			return e.Err
		case app.FrameEvent:
			gtx := app.NewContext(&ops, e)

			widgets.RenderTree(gtx, tree)

			// draw header

			//draw main window

			// background job
			// 	fetch file
			//	 process html
			//    	-> JOB: fetch and parse css
			//    	-> JOB: fetch and parse script
			//           -> compile script
			//           -> start script engine
			//    	-> style tree
			//     -> layout
			//         -> pass layout to render process
			//-> painting

			//TODO: render ui from html

			e.Frame(gtx.Ops)
		}
	}
}
