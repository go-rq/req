package tui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/go-rq/rq"
	"github.com/rivo/tview"
	"github.com/sahilm/fuzzy"
	"github.com/samber/lo"
)

type RequestSelect struct {
	app              *tview.Application
	list             *tview.List
	layout           *tview.Frame
	inputField       *tview.InputField
	selected         rq.Request
	searchString     string
	selectedCallback RequestSelectedCallback
	requests         []rq.Request
	previousView     View
}
type RequestSelectedCallback func(request rq.Request)

type requestFuzzySource []rq.Request

func (r requestFuzzySource) String(i int) string {
	return r[i].DisplayName()
}

func (r requestFuzzySource) Len() int {
	return len(r)
}

func NewRequestSelectView(app *tview.Application, requests []rq.Request, previousView View) *RequestSelect {
	view := &RequestSelect{
		app:          app,
		list:         tview.NewList(),
		inputField:   tview.NewInputField(),
		requests:     requests,
		previousView: previousView,
	}
	view.list.SetBorder(true).SetTitle("Requests")
	view.list.SetWrapAround(false)
	view.list.ShowSecondaryText(false)
	flex := tview.NewFlex().
		AddItem(tview.NewBox().SetBorder(false), 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(tview.NewBox().SetBorder(false), 0, 1, false).
			AddItem(view.list, 0, 10, false).
			AddItem(view.inputField, 1, 1, true).
			AddItem(tview.NewBox().SetBorder(false), 0, 1, false), 0, 4, false).
		AddItem(tview.NewBox().SetBorder(false), 0, 1, false)
	view.renderList(requests)
	view.layout = tview.NewFrame(flex)
	view.layout.AddText("Select Request", true, tview.AlignCenter, tcell.ColorBlue)
	view.layout.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEsc:
			view.previousView.Mount(view.app)
		case tcell.KeyUp, tcell.KeyDown, tcell.KeyPgUp, tcell.KeyPgDn, tcell.KeyHome, tcell.KeyEnd, tcell.KeyEnter:
			view.list.InputHandler()(event, nil)
		default:
			view.inputField.InputHandler()(event, nil)
			text := view.inputField.GetText()
			requests := view.requests
			if text != "" {
				matches := fuzzy.FindFrom(view.inputField.GetText(), requestFuzzySource(requests))
				requests = lo.Map(matches, func(m fuzzy.Match, _ int) rq.Request { return requests[m.Index] })
			}
			view.renderList(requests)
		}
		return event
	})
	view.inputField.SetLabel("Fuzzy Filter")
	return view
}

func (f *RequestSelect) SetCallback(callback RequestSelectedCallback) {
	f.selectedCallback = callback
}

func (f *RequestSelect) selectRequest(request rq.Request) func() {
	return func() {
		if f.selectedCallback != nil {
			f.clear()
			f.selectedCallback(request)
		}
	}
}

func (f *RequestSelect) clear() {
	f.inputField.SetText("")
	f.renderList(f.requests)
}

func (f *RequestSelect) renderList(requests []rq.Request) {
	f.list.Clear()
	for _, value := range requests {
		f.list.AddItem(value.DisplayName(), "", 0, f.selectRequest(value))
	}
}

func (f *RequestSelect) Mount(app *tview.Application) {
	app.SetRoot(f.layout, true)
}
