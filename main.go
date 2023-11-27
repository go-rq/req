package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/go-rq/req/internal/tui"
	"github.com/go-rq/rq"
	"github.com/rivo/tview"
)

var envFilePath string

func init() {
	initEnvFileFlags()
}

func main() {
	flag.Parse()
	app := tview.NewApplication()
	ctx := context.Background()
	if envFilePath != "" {
		env, err := loadEnvFile(envFilePath)
		if err != nil {
			panic(err)
		}
		ctx = rq.WithEnvironment(ctx, env)
	}
	path := "."
	if flag.NArg() > 1 {
		path = flag.Arg(0)
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
		rv := tui.NewRequestSelectView(app, path, prevView)
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

func initEnvFileFlags() {
	const usage = "path to .env file"
	flag.StringVar(&envFilePath, "env", "", usage)
	flag.StringVar(&envFilePath, "e", "", usage+" (shorthand)")
}

func loadEnvFile(path string) (map[string]string, error) {
	env := map[string]string{}
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(strings.TrimSpace(line), "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid line: %s", line)
		}
		env[parts[0]] = parts[1]
	}
	return env, nil
}
