package ui

import (
	//"log"
	//str "strings"

	//"github.com/gotk3/gotk3/gdk"
	"fmt"
	"log"

	"github.com/gotk3/gotk3/glib"
	"github.com/gotk3/gotk3/gtk"
	"github.com/snakesel/potbs_langui/pkg/gtkutils"
	"github.com/snakesel/potbs_langui/pkg/locales"
)

// IDs to access the tree view columns by
const (
	COLUMN_ID = iota
	COLUMN_ERROR
)

type VerifyWindow struct {
	Window *gtk.Window

	TreeView  *gtk.TreeView
	ListStore *gtk.ListStore

	LineSelection *gtk.TreeSelection

	BtnVerify *gtk.Button
	//BtnNew   *gtk.Button
	BtnExit       *gtk.Button
	boxChecks     *gtk.Box
	сhecksButtons []*gtk.CheckButton
}

func VerifyWindowNew() *VerifyWindow {
	var err error

	win := new(VerifyWindow)

	// Create a new toplevel window, set its title, and connect it to the
	// "destroy" signal to exit the GTK main loop when it is destroyed.
	win.Window, err = gtk.WindowNew(gtk.WINDOW_TOPLEVEL)
	checkErr(err, "Unable to create window")

	win.Window.SetTitle("Checking the translation to errors")
	win.Window.Connect("destroy", func() {
		gtk.MainQuit()
	})

	// Получаем остальные объекты MainWindow
	win.TreeView, err = gtk.TreeViewNew()
	checkErr(err)
	win.TreeView.AppendColumn(createTextColumn("ID", COLUMN_ID))
	win.TreeView.AppendColumn(createTextColumn("Error", COLUMN_ERROR))
	win.TreeView.SetFixedHeightMode(false) // режим фиксированной одинаковой высоты строк

	win.ListStore, err = gtk.ListStoreNew(glib.TYPE_STRING, glib.TYPE_STRING)
	checkErr(err)

	win.TreeView.SetModel(win.ListStore)

	win.LineSelection, err = win.TreeView.GetSelection()
	checkErr(err)
	win.LineSelection.SetMode(gtk.SELECTION_SINGLE)

	win.BtnExit, err = gtk.ButtonNew()
	checkErr(err)
	win.BtnExit.SetLabel("Exit")

	win.BtnExit.Connect("clicked", func() {
		win.Window.Destroy()
	})

	win.BtnVerify, err = gtk.ButtonNew()
	checkErr(err)
	win.BtnVerify.SetLabel("Verify")

	// построение UI
	scroll, err := gtk.ScrolledWindowNew(nil, nil)
	scroll.Add(win.TreeView)
	scroll.SetVExpand(true) //расширяемость по вертикали

	box, err := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 3)
	checkErr(err)
	boxButtons, err := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 3)
	checkErr(err)
	win.boxChecks, err = gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 0)
	checkErr(err)

	box.Add(win.boxChecks)
	box.Add(scroll)
	box.Add(boxButtons)

	// Кнопки
	boxButtons.Add(win.BtnVerify)
	//box2.Add(win.BtnClear)
	boxButtons.Add(win.BtnExit)

	boxButtons.SetHAlign(gtk.ALIGN_END) // расположение элементов по горизонтали
	boxButtons.SetSpacing(10)           // интервал между элементами
	boxButtons.SetHomogeneous(true)

	win.BtnVerify.SetHAlign(gtk.ALIGN_START)
	win.BtnExit.SetHAlign(gtk.ALIGN_END)
	//win.BtnNew.SetVisible(false)

	win.Window.Add(box)

	// Set the default window size.
	win.Window.SetDefaultSize(800, 600)

	return win
}

// Добавляем кнопку для проверки
func (win *VerifyWindow) AddCheckButton(label, tooltip string, active bool) error {

	button, err := gtk.CheckButtonNewWithLabel(label)
	if err != nil {
		return err
	}
	button.SetActive(active)
	button.SetTooltipText(tooltip)
	win.сhecksButtons = append(win.сhecksButtons, button)
	win.boxChecks.SetSpacing(len(win.сhecksButtons))
	win.boxChecks.Add(button)

	return nil

}

// Возвращаем значение кнопки по метке
func (win *VerifyWindow) GetCheckButtonActive(label string) (bool, error) {

	for _, check := range win.сhecksButtons {
		l, err := check.GetLabel()
		if err != nil {
			return false, err
		}
		if label == l {
			return check.GetActive(), nil
		}
	}

	return false, fmt.Errorf("Not found buton %s", label)

}

func (win *VerifyWindow) Run() {

	// Initialize GTK without parsing any command line arguments.
	//gtk.Init(nil)

	// Recursively show all widgets contained in this window.
	win.Window.ShowAll()

	//win.Window.SetPosition(gtk.WIN_POS_CENTER)

	//Begin executing the GTK main loop.  This blocks until
	//gtk.MainQuit() is run.
	//gtk.Main()

}

// Применение выбранного языка
func (win *VerifyWindow) SetLocale(locale *locales.Printer) {
	return
}

// Add a column to the tree view (during the initialization of the tree view)
// We need to distinct the type of data shown in either column.
func createTextColumn(title string, id int) *gtk.TreeViewColumn {
	// In this column we want to show text, hence create a text renderer
	cellRenderer, err := gtk.CellRendererTextNew()
	if err != nil {
		log.Fatal("Unable to create text cell renderer:", err)
	}

	// Tell the renderer where to pick input from. Text renderer understands
	// the "text" property.
	column, err := gtk.TreeViewColumnNewWithAttribute(title, cellRenderer, "text", id)
	if err != nil {
		log.Fatal("Unable to create cell column:", err)
	}

	return column
}

// Append a row to the list store for the tree view
func (win *VerifyWindow) AddRow(id, text string) error {
	// Get an iterator for a new row at the end of the list store
	iter := win.ListStore.Append()

	// Set the contents of the list store row that the iterator represents
	err := win.ListStore.Set(iter,
		[]int{COLUMN_ID, COLUMN_ERROR},
		[]interface{}{id, text})
	if err != nil {
		log.Fatal("[ERR]\tUnable to add row:", err)
	}

	return err

}

// Возвращает ID выбранной записи
func (win *VerifyWindow) GetSelectedID() string {
	_, iter, ok := win.LineSelection.GetSelected()
	if !ok {
		log.Println("[err]\tGetSelected error iter")
		return ""
	}

	id, err := gtkutils.GetListStoreValueString(win.ListStore, iter, COLUMN_ID)
	if err != nil {
		return ""
	}

	return id
}

func checkErr(e error, text_opt ...string) {
	if e != nil {

		if len(text_opt) > 0 {
			log.Println(text_opt[0])
		}
		// panic for any errors.
		//log.Panic(e)
		log.Fatal(e, e.Error())
	}
}
