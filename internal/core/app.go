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
		<style>
			* { font-size: 22px; }
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
			span { display: inline; background-color: gray; }
			button { display: block; height: 50px; width: 50px; background-color: maroon; }
			button:hover { background-color: blue; }
			input { display: block; height: 40px; width: 100px; background-color: green; }
		</style>
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
		<button>A</button>
		<input/>
		<div>
			Text Before <span>Some Text here</span>
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

				if err := mainFrame.Resize(width, height); err != nil {
					logger.ErrorContext(ctx, "failed to resize", slog.String("error", err.Error()))
				}
			}
		case app.FrameEvent:
			gtx := app.NewContext(&ops, e)

			mainFrame.Render(gtx)

			e.Frame(gtx.Ops)
		}
	}
}
