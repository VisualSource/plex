package core

import (
	"context"
	"log/slog"
	"strings"

	"gioui.org/app"
	"gioui.org/op"
)

func StartPlex(ctx context.Context, logger *slog.Logger, window *app.Window, remoteDebuggingPort int) error {
	width := 720
	height := 480

	mainFrame := NewFrame(logger, ctx)
	mainFrame.width = width
	mainFrame.height = height

	if err := mainFrame.Load(strings.NewReader(`<!DOCTYPE html>
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
	`)); err != nil {
		return err
	}

	var ops op.Ops
	for {
		switch e := window.Event().(type) {
		case app.DestroyEvent:
			return e.Err

		case app.ConfigEvent:
			size := e.Config.Size
			if height != size.Y || width != size.X {
				logger.DebugContext(ctx, "resize", slog.Int("width", size.X), slog.Int("height", size.Y))
				height = size.Y
				width = size.X

				mainFrame.Resize(width, height)
				// todo on resize event a resize event are rerender styletree (apply styles in media queries) -> layout tree
			}
		case app.FrameEvent:
			gtx := app.NewContext(&ops, e)

			mainFrame.Render(gtx)

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
