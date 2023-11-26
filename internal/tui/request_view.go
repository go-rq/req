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

var SelectedRequest *rq.Request

type RequestView struct {
	app          *tview.Application
	request      *rq.Request
	layout       *tview.Flex
	frame        *tview.Frame
	main         *tview.TextView
	commandsView *tview.TextView
	context      context.Context
	previousView View
	commands     []Command
	responses    []Response
	baseCommands []Command
}

type Response struct {
	rq.Response
	cachedPrettyString string
	cachedRawString    string
}

func (p *Response) prettyString() (string, error) {
	if p.cachedPrettyString == "" {
		text, err := p.Response.PrettyString()
		if err != nil {
			return "", err
		}
		p.cachedPrettyString = colorize(text, "base16-snazzy", true)
	}
	return p.cachedPrettyString, nil
}

func (p *Response) rawString() string {
	if p.cachedRawString == "" {
		p.cachedRawString = colorize(p.Response.String(), "base16-snazzy", true)
	}

	return p.cachedRawString
}

func NewRequestView(ctx context.Context, app *tview.Application, request rq.Request, previousView View) *RequestView {
	flex := tview.NewFlex()
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
	}

	view.frame.AddText(request.DisplayName(), true, tview.AlignCenter, tcell.ColorForestGreen)
	view.layout.SetDirection(tview.FlexRow)
	view.layout.AddItem(view.main, 0, 7, true).
		AddItem(view.commandsView, 1, 0, false)
	view.showRequest()
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

func colorize(text string, theme string, color bool) string {
	if !color {
		return text
	}
	buf := bytes.NewBuffer(nil)
	quick.Highlight(buf, text, "HTTP", "terminal8", theme)
	return tview.TranslateANSI(buf.String())
}

func (view *RequestView) showRequest() {
	view.main.SetBorder(false).SetTitle("Request").SetTitleColor(tcell.ColorAliceBlue)
	view.main.SetTextColor(tcell.ColorDefault)
	view.main.SetText(colorize(view.request.HttpText(), "doom-one", true))
	commands := []Command{
		{
			Name: "Back",
			Key:  tcell.KeyEscape,
			Handler: func() {
				view.previousView.Mount(view.app)
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
	}
	view.registerCommands(append(view.baseCommands, commands...)...)
}

func (view *RequestView) showPrettyResponse(idx int) {
	resp := &view.responses[idx]
	view.main.SetBorder(false).SetTitle("Response (Pretty)").SetTitleColor(tcell.ColorLawnGreen)
	view.main.SetTextColor(tcell.ColorDefault)
	text, err := resp.prettyString()
	if err != nil {
		view.showError(err)
		return
	}
	view.main.SetText(text)
	commands := []Command{
		{
			Name: "Clear",
			Key:  tcell.KeyEscape,
			Handler: func() {
				view.showRequest()
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
	}
	view.registerCommands(append(view.baseCommands, commands...)...)
}

func (view *RequestView) showRawResponse(idx int) {
	resp := &view.responses[idx]
	view.main.SetBorder(false).SetTitle("Response (Raw)").SetTitleColor(tcell.ColorLawnGreen)
	view.main.SetTextColor(tcell.ColorDefault)
	text := resp.rawString()
	view.main.SetText(text)
	commands := []Command{
		{
			Name: "Clear",
			Key:  tcell.KeyEscape,
			Handler: func() {
				view.showRequest()
			},
		},
		{
			Name: "Pretty Print",
			Key:  tcell.KeyRune,
			Rune: 'p',
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
	}
	view.registerCommands(append(view.baseCommands, commands...)...)
}

func (view *RequestView) showError(err error) {
	view.main.SetBorder(false).SetTitle("Error!").SetTitleColor(tcell.ColorOrangeRed)
	view.main.SetText(err.Error())
	view.main.SetTextColor(tcell.ColorOrangeRed)
	commands := []Command{
		{
			Name: "Clear",
			Key:  tcell.KeyEscape,
			Handler: func() {
				view.showRequest()
			},
		},
	}
	view.registerCommands(append(view.baseCommands, commands...)...)
}

func (view *RequestView) showAssertions(idx int, previousView func(int)) {
	resp := view.responses[idx]
	view.main.SetBorder(false).SetTitle("Assertions").SetTitleColor(tcell.ColorYellowGreen)
	view.main.SetDynamicColors(true)

	builder := strings.Builder{}
	builder.WriteString("[::bu]Pre-Request Assertions[::-]:\n")
	writeAssertionResults(&builder, resp.PreRequestAssertions...)
	builder.WriteString("\n[::bu]Post-Request Assertions[::-]:\n")
	writeAssertionResults(&builder, resp.PostRequestAssertions...)

	view.main.SetText(builder.String())
	commands := []Command{
		{
			Name: "Clear",
			Key:  tcell.KeyEscape,
			Handler: func() {
				previousView(idx)
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
	app.SetRoot(view.frame, true)
}
