package core

import (
	"context"
	"log/slog"
	"time"

	"gioui.org/app"
	"gioui.org/op"
	"github.com/VisualSource/plex/internal/layouts/widgets"
)

func StartPlex(ctx context.Context, logger *slog.Logger, window *app.Window) error {
	startTime := time.Now()
	tree := renderHtml(`<!DOCTYPE html>
	<html lang="en">
	<head>
		<meta charset="UTF-8">
		<meta name="viewport" content="width=device-width, initial-scale=1.0">
		<title>Document</title>
	</head>
	<body>
		<div class="a">
			<div class="b">
				<div class="c">
					<div class="d">
						<div class="e">
						<div class="f">
							<div class="g">
							</div>
						</div>
						</div>
					</div>
				</div>
			</div>
		</div>
	</body>
	</html>
	`, `
	div { display: block; padding-left: 12px; padding-right: 12px; padding-top: 12px; padding-bottom: 12px; }
	head { display: none; background-color: gray; }
	html { display: block; background-color: maroon; }
	body { display: block; background-color: coral; }
	.a { background-color: #ff0000; }
	.b { background-color: #ffa500; }
	.c { background-color: #ffff00; }
	.d { background-color: #008000; }
	.e { background-color: #0000ff; }
	.f { background-color: #4b0082; }
	.g { background-color: #800080; }
	`, 854, 480)

	elapsed := time.Since(startTime)

	logger.DebugContext(ctx, "html -> css -> cssom -> styletree -> layout: time", slog.String("time", elapsed.String()))

	var ops op.Ops
	for {
		switch e := window.Event().(type) {
		case app.DestroyEvent:
			return e.Err

		case app.ConfigEvent:
			// todo on resize event a resize event are rerender styletree (apply styles in media queries) -> layout tree

			size := e.Config.Size

			logger.DebugContext(ctx, "resize", slog.Int("width", size.X), slog.Int("height", size.Y))

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
