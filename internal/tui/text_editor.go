package tui

import (
	"context"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type TextEditorView struct {
	app          *tview.Application
	layout       *tview.Grid
	frame        *tview.Frame
	main         *tview.TextArea
	helpInfo     *tview.TextView
	context      context.Context
	previousView View
}

func NewTextEditorView(ctx context.Context, app *tview.Application, cancelFunc func(), saveFunc func(string), title, text string) *TextEditorView {
	grid := tview.NewGrid()
	helpInfo := tview.NewTextView()
	view := &TextEditorView{
		app:      app,
		context:  ctx,
		main:     tview.NewTextArea(),
		helpInfo: helpInfo,
		layout:   grid,
	}
	view.main.SetPlaceholder("Enter your environment variables here using the format KEY=VALUE separated by new lines")
	view.main.SetTitle("Editor")
	view.main.SetBorder(true)
	view.layout.SetRows(0, 2)
	view.layout.
		AddItem(view.main, 0, 0, 1, 2, 0, 0, true)
	view.layout.
		AddItem(helpInfo, 1, 0, 1, 2, 0, 0, false)
	helpInfo.SetText(`Ctrl+S: Save, Esc: Cancel`)
	helpInfo.SetTextAlign(tview.AlignCenter)

	view.main.SetText(text, true)
	view.main.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyCtrlS {
			saveFunc(view.main.GetText())
		}
		if event.Key() == tcell.KeyEsc {
			cancelFunc()
		}
		return event
	})
	view.frame = tview.NewFrame(view.layout)
	view.frame.SetTitle(title)
	view.frame.AddText(title, true, tview.AlignCenter, tcell.ColorForestGreen)
	return view
}

func (view *TextEditorView) Mount(app *tview.Application) {
	app.SetRoot(view.frame, true)
}
