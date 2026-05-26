package core

import (
	"context"
	"log/slog"

	"github.com/Zyko0/go-sdl3/bin/binimg"
	"github.com/Zyko0/go-sdl3/bin/binsdl"
	"github.com/Zyko0/go-sdl3/bin/binttf"
	"github.com/Zyko0/go-sdl3/sdl"
	"github.com/Zyko0/go-sdl3/ttf"
)

func StartPlex(logger *slog.Logger) error {
	defer binsdl.Load().Unload()
	defer binttf.Load().Unload()
	defer binimg.Load().Unload()
	defer sdl.Quit()

	ctx := context.Background()

	if err := sdl.Init(sdl.INIT_VIDEO); err != nil {
		return err
	}
	if err := ttf.Init(); err != nil {
		return err
	}

	window, renderer, err := sdl.CreateWindowAndRenderer("Plex", 800, 480, sdl.WINDOW_RESIZABLE)
	if err != nil {
		return err
	}

	defer renderer.Destroy()
	defer window.Destroy()

	renderer.SetDrawColor(255, 255, 255, 255)

	logger.DebugContext(ctx, "SDL", slog.String("version", sdl.GetVersion().String()))

	sdl.RunLoop(func() error {
		var event sdl.Event

		for sdl.PollEvent(&event) {
			switch event.Type {
			case sdl.EVENT_QUIT:
				return sdl.EndLoop
			}
		}

		renderer.DebugText(50, 50, "Hello, From SDL3")

		// background job
		// 	fetch file
		//	 process html
		//    	-> JOB: fetch and parse css
		//    	-> JOB: fetch and parse script
		//           -> compile script
		//           -> start script engine
		//    	-> link html and css
		//     -> layout
		//         -> pass layout to render process
		//-> render

		renderer.Present()
		return nil
	})

	return nil
}
