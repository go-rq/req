package main

import (
	"context"
	"os"

	"github.com/go-rq/req/internal/tui"
	"github.com/go-rq/rq"
	"github.com/rivo/tview"
)

func main() {
	app := tview.NewApplication()
	ctx := context.Background()
	path := "."
	if len(os.Args) > 1 {
		path = os.Args[1]
	}
	fileSelectView := tui.NewFileSelectView(path)
	fileSelectView.SetCallback(selectFile(ctx, app, fileSelectView))
	fileSelectView.Mount(app)
	if err := app.Run(); err != nil {
		panic(err)
	}
}

func selectFile(ctx context.Context, app *tview.Application, prevView tui.View) func(string) {
	return func(path string) {
		reqs, err := rq.ParseFromFile(path)
		if err != nil {
			panic(err)
		}
		rv := tui.NewRequestSelectView(app, reqs, prevView)
		rv.SetCallback(selectRequest(ctx, app, rv))
		rv.Mount(app)
	}
}

func selectRequest(ctx context.Context, app *tview.Application, prevView tui.View) func(request rq.Request) {
	return func(request rq.Request) {
		rv := tui.NewRequestView(ctx, app, request, prevView)
		rv.Mount(app)
	}
}
