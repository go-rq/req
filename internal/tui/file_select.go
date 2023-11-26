package tui

import (
	"os"
	"path/filepath"
	"regexp"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/sahilm/fuzzy"
	"github.com/samber/lo"
)

var (
	httpFileFilter = regexp.MustCompile(`^.*\.http$`)
	envFileFilter  = regexp.MustCompile(`^.*\.env$`)
)

type View interface {
	Mount(app *tview.Application)
}

type FileSelect struct {
	app              *tview.Application
	list             *tview.List
	layout           *tview.Frame
	inputField       *tview.InputField
	selectedFile     string
	path             string
	searchString     string
	selectedCallback FileSelectedCallback
	files            []string
}
type FileSelectedCallback func(string)

func NewFileSelectView(path string) *FileSelect {
	view := &FileSelect{
		path:       path,
		list:       tview.NewList(),
		inputField: tview.NewInputField(),
		files:      loadFiles(path, httpFileFilter),
	}
	view.list.SetBorder(true).SetTitle("Files")
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
	view.listFiles(httpFileFilter)
	view.layout = tview.NewFrame(flex)
	view.layout.AddText("Select File", true, tview.AlignCenter, tcell.ColorBlue)
	view.layout.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEsc, tcell.KeyUp, tcell.KeyDown, tcell.KeyPgUp, tcell.KeyPgDn, tcell.KeyHome, tcell.KeyEnd, tcell.KeyEnter:
			view.list.InputHandler()(event, nil)
		default:
			view.inputField.InputHandler()(event, nil)
			text := view.inputField.GetText()
			files := view.files
			if text != "" {
				matches := fuzzy.Find(view.inputField.GetText(), view.files)
				files = lo.Map(matches, func(m fuzzy.Match, _ int) string { return m.Str })
			}
			view.renderList(files)
		}
		return event
	})
	view.inputField.SetLabel("Fuzzy Filter")
	return view
}

func (f *FileSelect) SetCallback(callback FileSelectedCallback) {
	f.selectedCallback = callback
}

func (f *FileSelect) listFiles(filter *regexp.Regexp) {
	f.files = loadFiles(f.path, filter)
	f.renderList(f.files)
}

func (f *FileSelect) selectFile(path string) func() {
	return func() {
		f.selectedFile = path
		if f.selectedCallback != nil {
			f.clear()
			f.selectedCallback(path)
		}
	}
}

func (f *FileSelect) clear() {
	f.inputField.SetText("")
	f.renderList(f.files)
}

func (f *FileSelect) renderList(values []string) {
	f.list.Clear()
	for _, value := range values {
		f.list.AddItem(value, "", 0, f.selectFile(value))
	}
}

func (f *FileSelect) Mount(app *tview.Application) {
	app.SetRoot(f.layout, true)
}

func loadFiles(path string, filter *regexp.Regexp) []string {
	var files []string
	filepath.Walk(path, func(path string, _ os.FileInfo, _ error) error {
		if filter.MatchString(path) {
			files = append(files, path)
		}
		return nil
	})
	return files
}
