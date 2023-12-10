package tui

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/alecthomas/chroma/quick"
	"github.com/atotto/clipboard"
	"github.com/gdamore/tcell/v2"
	"github.com/go-rq/rq"
	"github.com/rivo/tview"
)

const (
	HTTPLexer        = "HTTP"
	JavascriptLexer  = "javascript"
	DefaultTheme     = "base16-snazzy"
	AlternativeTheme = "doom-one"
	ColorScheme      = "terminal16"
)

var SelectedRequest *rq.Request

type RequestView struct {
	app            *tview.Application
	request        *rq.Request
	layout         *tview.Flex
	frame          *tview.Frame
	main           *tview.TextView
	commandsView   *tview.TextView
	context        context.Context
	previousView   View
	refreshContent func()
	commands       []Command
	responses      []Response
	baseCommands   []Command
}

type Response struct {
	cachedPrettyString string
	cachedRawString    string
	rq.Response
}

func (p *Response) prettyString() (string, error) {
	if p.cachedPrettyString == "" {
		text, err := p.Response.PrettyString()
		if err != nil {
			return "", err
		}
		p.cachedPrettyString = colorize(text, HTTPLexer, DefaultTheme, true)
	}
	return p.cachedPrettyString, nil
}

func (p *Response) rawString() string {
	if p.cachedRawString == "" {
		p.cachedRawString = colorize(p.Response.String(), HTTPLexer, DefaultTheme, true)
	}

	return p.cachedRawString
}

func NewRequestView(ctx context.Context, app *tview.Application, request rq.Request, previousView View) *RequestView {
	flex := tview.NewFlex().SetDirection(tview.FlexRow)
	view := &RequestView{
		app:          app,
		context:      ctx,
		request:      &request,
		frame:        tview.NewFrame(flex),
		main:         tview.NewTextView().SetDynamicColors(true).SetRegions(true),
		commandsView: tview.NewTextView(),
		layout:       flex,
		previousView: previousView,
	}
	view.baseCommands = []Command{
		{
			Name: "Quit",
			Key:  tcell.KeyCtrlQ,
			Handler: func() {
				app.Stop()
			},
		},
		{
			Name: "Copy to Clipboard",
			Key:  tcell.KeyRune,
			Rune: 'c',
			Handler: func() {
				clipboard.WriteAll(view.main.GetText(true))
			},
		},
		{
			Name: "Variables",
			Key:  tcell.KeyRune,
			Rune: 'v',
			Handler: func() {
				ev := NewTextEditorView(
					ctx,
					app,
					func() { view.Mount(app) },
					func(update string) {
						view.setEnvironment(ctx, update)
						view.Mount(app)
					},
					"Variables",
					getEnvironmentText(ctx))
				ev.Mount(app)
			},
		},
	}

	view.frame.AddText(request.DisplayName(), true, tview.AlignCenter, tcell.ColorForestGreen)
	view.layout.AddItem(view.main, 0, 7, true).
		AddItem(view.commandsView, 1, 0, false)
	view.showRawRequest()
	view.main.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		for _, cmd := range view.commands {
			if event.Key() == cmd.Key {
				if cmd.Key != tcell.KeyRune || event.Rune() == cmd.Rune {
					cmd.Handler()
					return event
				}
			}
		}
		return event
	})
	return view
}

type Command struct {
	Handler func()
	Name    string
	Rune    rune
	Key     tcell.Key
}

func colorize(text, lexer, theme string, color bool) string {
	if !color {
		return text
	}
	buf := bytes.NewBuffer(nil)
	quick.Highlight(buf, text, lexer, ColorScheme, theme)
	return tview.TranslateANSI(buf.String())
}

func (view *RequestView) showRawRequest() {
	view.main.SetBorder(false).SetTitle("Request").SetTitleColor(tcell.ColorAliceBlue)
	view.main.SetTextColor(tcell.ColorDefault)
	view.refreshContent = func() {
		view.main.SetText(colorize(view.request.HttpText(), HTTPLexer, AlternativeTheme, true))
	}
	view.refreshContent()
	commands := []Command{
		{
			Name: "Back",
			Key:  tcell.KeyEscape,
			Handler: func() {
				view.previousView.Mount(view.app)
			},
		},
		{
			Name: "With Environment",
			Key:  tcell.KeyTab,
			Handler: func() {
				view.showProcessedRequest()
			},
		},
		{
			Name: "Scripts",
			Key:  tcell.KeyRune,
			Rune: 's',
			Handler: func() {
				view.showScripts(view.showRawRequest)
			},
		},
		{
			Name: "Send",
			Key:  tcell.KeyEnter,
			Handler: func() {
				resp, err := view.request.Do(view.context)
				if err != nil {
					view.showError(err)
					return
				}
				view.responses = append(view.responses, Response{Response: *resp})
				view.showPrettyResponse(len(view.responses) - 1)
			},
		},
		{
			Name: "Logs",
			Key:  tcell.KeyRune,
			Rune: 'l',
			Handler: func() {
				view.showLogs(view.showRawRequest)
			},
		},
	}
	view.registerCommands(append(view.baseCommands, commands...)...)
}

func (view *RequestView) showProcessedRequest() {
	view.main.SetBorder(false).SetTitle("Request").SetTitleColor(tcell.ColorAliceBlue)
	view.main.SetTextColor(tcell.ColorDefault)
	view.refreshContent = func() {
		request := view.request.ApplyEnv(view.context)
		view.main.SetText(colorize(request.HttpText(), HTTPLexer, "doom-one", true))
	}
	view.refreshContent()
	commands := []Command{
		{
			Name: "Back",
			Key:  tcell.KeyEscape,
			Handler: func() {
				view.previousView.Mount(view.app)
			},
		},
		{
			Name: "Raw Request",
			Key:  tcell.KeyTab,
			Handler: func() {
				view.showRawRequest()
			},
		},
		{
			Name: "Scripts",
			Key:  tcell.KeyRune,
			Rune: 's',
			Handler: func() {
				view.showScripts(view.showRawRequest)
			},
		},
		{
			Name: "Send",
			Key:  tcell.KeyEnter,
			Handler: func() {
				resp, err := view.request.Do(view.context)
				if err != nil {
					view.showError(err)
					return
				}
				view.responses = append(view.responses, Response{Response: *resp})
				view.showPrettyResponse(len(view.responses) - 1)
			},
		},
		{
			Name: "Logs",
			Key:  tcell.KeyRune,
			Rune: 'l',
			Handler: func() {
				view.showLogs(view.showRawRequest)
			},
		},
	}
	view.registerCommands(append(view.baseCommands, commands...)...)
}

func (view *RequestView) showPrettyResponse(idx int) {
	view.main.SetBorder(false).SetTitle("Response (Pretty)").SetTitleColor(tcell.ColorLawnGreen)
	view.main.SetTextColor(tcell.ColorDefault)
	view.refreshContent = func() {
		resp := &view.responses[idx]
		text, err := resp.prettyString()
		if err != nil {
			view.showError(err)
			return
		}
		view.main.SetText(text)
	}
	view.refreshContent()
	commands := []Command{
		{
			Name: "Clear",
			Key:  tcell.KeyEscape,
			Handler: func() {
				view.showRawRequest()
			},
		},
		{
			Name: "Raw",
			Key:  tcell.KeyRune,
			Rune: 'r',
			Handler: func() {
				view.showRawResponse(idx)
			},
		},
		{
			Name: "Assertions",
			Key:  tcell.KeyRune,
			Rune: 'a',
			Handler: func() {
				view.showAssertions(idx, view.showPrettyResponse)
			},
		},
		{
			Name: "Logs",
			Key:  tcell.KeyRune,
			Rune: 'l',
			Handler: func() {
				view.showLogs(func() { view.showPrettyResponse(idx) })
			},
		},
	}
	view.registerCommands(append(view.baseCommands, commands...)...)
}

func (view *RequestView) showRawResponse(idx int) {
	view.main.SetBorder(false).SetTitle("Response (Raw)").SetTitleColor(tcell.ColorLawnGreen)
	view.main.SetTextColor(tcell.ColorDefault)
	view.refreshContent = func() {
		resp := &view.responses[idx]
		text := resp.rawString()
		view.main.SetText(text)
	}
	view.refreshContent()
	commands := []Command{
		{
			Name: "Clear",
			Key:  tcell.KeyEscape,
			Handler: func() {
				view.showRawRequest()
			},
		},
		{
			Name: "Pretty Print",
			Key:  tcell.KeyRune,
			Rune: 'r',
			Handler: func() {
				view.showPrettyResponse(idx)
			},
		},
		{
			Name: "Assertions",
			Key:  tcell.KeyRune,
			Rune: 'a',
			Handler: func() {
				view.showAssertions(idx, view.showRawResponse)
			},
		},
		{
			Name: "Logs",
			Key:  tcell.KeyRune,
			Rune: 'l',
			Handler: func() {
				view.showLogs(func() { view.showRawResponse(idx) })
			},
		},
	}
	view.registerCommands(append(view.baseCommands, commands...)...)
}

func (view *RequestView) showError(err error) {
	view.main.SetBorder(false).SetTitle("Error!").SetTitleColor(tcell.ColorOrangeRed)
	view.refreshContent = func() {
		view.main.SetText(err.Error())
	}
	view.refreshContent()
	view.main.SetTextColor(tcell.ColorOrangeRed)
	commands := []Command{
		{
			Name: "Clear",
			Key:  tcell.KeyEscape,
			Handler: func() {
				view.showRawRequest()
			},
		},
		{
			Name: "Logs",
			Key:  tcell.KeyRune,
			Rune: 'l',
			Handler: func() {
				view.showLogs(func() { view.showError(err) })
			},
		},
	}
	view.registerCommands(append(view.baseCommands, commands...)...)
}

func (view *RequestView) showAssertions(idx int, previousView func(int)) {
	view.main.SetBorder(false).SetTitle("Assertions").SetTitleColor(tcell.ColorYellowGreen)
	view.main.SetDynamicColors(true)

	view.refreshContent = func() {
		resp := view.responses[idx]
		builder := strings.Builder{}
		builder.WriteString("[::bu]Pre-Request Assertions[::-]:\n")
		writeAssertionResults(&builder, view.request.PreRequestAssertions...)
		builder.WriteString("\n[::bu]Post-Request Assertions[::-]:\n")
		writeAssertionResults(&builder, resp.PostRequestAssertions...)
		view.main.SetText(builder.String())
	}
	view.refreshContent()
	commands := []Command{
		{
			Name: "Clear",
			Key:  tcell.KeyEscape,
			Handler: func() {
				previousView(idx)
			},
		}, {
			Name: "Logs",
			Key:  tcell.KeyRune,
			Rune: 'l',
			Handler: func() {
				view.showLogs(func() { view.showAssertions(idx, previousView) })
			},
		},
	}
	view.registerCommands(append(view.baseCommands, commands...)...)
}

func (view *RequestView) showScripts(previousView func()) {
	view.main.SetBorder(false).SetTitle("Assertions").SetTitleColor(tcell.ColorYellowGreen)
	view.main.SetDynamicColors(true)

	view.refreshContent = func() {
		builder := strings.Builder{}
		builder.WriteString("[::bu]Pre-Request Scripts[::-]:\n")
		builder.WriteString(colorize(view.request.PreRequestScript, JavascriptLexer, AlternativeTheme, true) + "\n")
		builder.WriteString("\n[::bu]Post-Request Scripts[::-]:\n")
		builder.WriteString(colorize(view.request.PostRequestScript, JavascriptLexer, AlternativeTheme, true))
		view.main.SetText(builder.String())
	}
	view.refreshContent()
	commands := []Command{
		{
			Name: "Clear",
			Key:  tcell.KeyEscape,
			Handler: func() {
				previousView()
			},
		},
	}
	view.registerCommands(append(view.baseCommands, commands...)...)
}

func (view *RequestView) showLogs(previousView func()) {
	view.main.SetBorder(false).SetTitle("Logs").SetTitleColor(tcell.ColorViolet)
	view.main.SetDynamicColors(true)

	view.refreshContent = func() {
		view.main.SetText(strings.Join(view.request.Logs, "\n"))
	}
	view.refreshContent()
	commands := []Command{
		{
			Name: "Clear",
			Key:  tcell.KeyEscape,
			Handler: func() {
				previousView()
			},
		},
	}
	view.registerCommands(append(view.baseCommands, commands...)...)
}

func writeAssertionResults(writer io.Writer, assertions ...rq.Assertion) {
	for _, assertion := range assertions {
		color := "green"
		statusText := "passed"
		if !assertion.Success {
			statusText = "FAILED"
			color = "red"
		}
		status := fmt.Sprintf("[%s]%s", color, statusText)
		fmt.Fprintf(writer, "-- [%s[-]]: %s\n", status, assertion.Message)
	}
}

func (view *RequestView) registerCommands(commands ...Command) {
	var parts []string
	for _, command := range commands {
		if command.Rune != 0 {
			parts = append(parts, fmt.Sprintf("%s (%s)", command.Name, string(command.Rune)))
			continue
		}
		parts = append(parts, fmt.Sprintf("%s (%s)", command.Name, tcell.KeyNames[command.Key]))

	}
	view.commandsView.SetText(strings.Join(parts, " | "))
	view.commands = commands
}

func (view *RequestView) Mount(app *tview.Application) {
	view.refreshContent()
	app.SetRoot(view.frame, true)
}

func (view *RequestView) setEnvironment(ctx context.Context, text string) {
	env := rq.GetEnvironment(ctx)
	clear(env)
	lines := strings.Split(text, "\n")
	for _, line := range lines {
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		env[parts[0]] = parts[1]
	}
}

func getEnvironmentText(ctx context.Context) string {
	env := rq.GetEnvironment(ctx)
	builder := strings.Builder{}
	lineWritten := false
	for k, v := range env {
		if lineWritten {
			builder.WriteString("\n")
		}
		builder.WriteString(fmt.Sprintf("%s=%s", k, v))
		lineWritten = true
	}
	return builder.String()
}
