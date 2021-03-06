package tmpl

import (
	"bufio"
	"fmt"
	"log"
	"os"
	str "strings"

	"github.com/snakesel/potbs_langui/pkg/gtkutils"
	"github.com/snakesel/potbs_langui/pkg/locales"

	"github.com/gotk3/gotk3/gtk"
)

var dialog *DialogWindow

const (
	separator = "<:>"
	TmplGlade = "data/ui/tmpl.glade"
)

type TTmpl struct {
	En string
	Ru string
}

const (
	columnEN = iota
	columnRU
)

type TmplWindow struct {
	Window *gtk.Window

	Label_Head *gtk.Label
	TreeView   *gtk.TreeView
	ListStore  *gtk.ListStore

	LineSelection *gtk.TreeSelection

	BtnClose *gtk.Button
	BtnSave  *gtk.Button
	BtnAdd   *gtk.Button
	BtnDel   *gtk.Button

	Iterator *gtk.TreeIter

	Col_SourceLang *gtk.TreeViewColumn
	Col_TargetLang *gtk.TreeViewColumn
	locale         *locales.Printer
}

type DialogWindow struct {
	Window *gtk.Dialog

	TextEn *gtk.Entry
	TextRu *gtk.Entry

	BtnCancel *gtk.Button
	BtnOk     *gtk.Button

	NewItem bool
}

func TmplWindowCreate() *TmplWindow {

	// Создаём билдер
	b, err := gtk.BuilderNewFromFile(TmplGlade)
	errorCheck(err, "Error: No load tmpl.glade")

	win := new(TmplWindow)

	// Получаем объект главного окна по ID
	obj, err := b.GetObject("window_tmpl")
	errorCheck(err, "Error: No find window_tmpl")

	win.Window = obj.(*gtk.Window)

	// Получаем остальные объекты window_main
	win.Label_Head = gtkutils.GetLabel(b, "label_header")
	win.TreeView = gtkutils.GetTreeView(b, "tmpl_treeview")
	win.ListStore = gtkutils.GetListStore(b, "tmpl_liststore")
	win.LineSelection = gtkutils.GetTreeSelection(b, "tmpl_lineSelection")

	win.BtnClose = gtkutils.GetButton(b, "tmpl_btn_close")
	win.BtnSave = gtkutils.GetButton(b, "tmpl_btn_save")
	win.BtnAdd = gtkutils.GetButton(b, "tmpl_btn_add")
	win.BtnDel = gtkutils.GetButton(b, "tmpl_btn_del")
	win.Col_SourceLang = gtkutils.GetTreeViewColumn(b, "tmpl_Col_SourceLang")
	win.Col_TargetLang = gtkutils.GetTreeViewColumn(b, "tmpl_Col_TargetLang")

	win.BtnClose.Connect("clicked", func() {
		win.Window.Close()
	})

	win.TreeView.Connect("row-activated", func() {
		win.lineSelected()
	})

	win.BtnAdd.Connect("clicked", func() {
		dialog.NewItem = true
		dialog.TextEn.SetText("")
		dialog.TextRu.SetText("")
		dialog.Window.Run()
	})

	win.BtnDel.Connect("clicked", func() {
		_, win.Iterator, _ = win.LineSelection.GetSelected()
		win.ListStore.Remove(win.Iterator)
	})

	return win
}

func dialogWindowCreate() *DialogWindow {

	// Создаём билдер
	b, err := gtk.BuilderNewFromFile(TmplGlade)
	errorCheck(err, "Error: No load tmpl.glade")

	// Окно диалога
	dialog := new(DialogWindow)

	obj, err := b.GetObject("tmpl_dialog")
	errorCheck(err)
	dialog.Window = obj.(*gtk.Dialog)
	dialog.Window.Connect("close", func() {
		dialog.Window.Hide()
	})
	dialog.Window.Connect("destroy", func() {
		dialog.Window.Hide()
	})

	//Убираем кнопку "Закрыть(X)"
	dialog.Window.SetDeletable(false)

	// Получаем остальные объекты dialog_translite
	dialog.TextEn = gtkutils.GetEntry(b, "tmpl_dlg_entry_en")
	dialog.TextRu = gtkutils.GetEntry(b, "tmpl_dlg_entry_ru")

	dialog.BtnCancel = gtkutils.GetButton(b, "tmpl_dlg_btn_cancel")
	dialog.BtnOk = gtkutils.GetButton(b, "tmpl_dlg_btn_ok")

	//Сигналы
	dialog.BtnCancel.Connect("clicked", func() {
		dialog.Window.Hide()
	})

	return dialog
}

func LoadTmplFromFile(patch string) []TTmpl {
	Tmpls := make([]TTmpl, 0)

	if _, err := os.Stat(patch); err != nil {
		if os.IsNotExist(err) {
			log.Printf("[INFO]\tФайл шаблонов %s не найден.\n", patch)
			return Tmpls
		} else {
			log.Println("[ERR]\tНеизвестная ошибки при открытии шаблонов")
			return nil
		}

	}

	file, err := os.Open(patch)
	errorCheck(err)
	defer file.Close()

	var splitline []string
	var line TTmpl
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		splitline = str.SplitN(scanner.Text(), separator, 2)
		line.En = splitline[0]
		line.Ru = splitline[1]
		Tmpls = append(Tmpls, line)
	}

	log.Println("[INFO]\tШаблонов загружено: ", len(Tmpls))
	return Tmpls
}

func SaveTmplToFile(tmpls []TTmpl, patch string) {

	file, err := os.Create(patch)
	errorCheck(err, "Unable to create file: "+patch)
	defer file.Close()

	for _, line := range tmpls {
		file.WriteString(fmt.Sprintf("%s%s%s\n", line.En, separator, line.Ru))
	}
}

func addRow(listStore *gtk.ListStore, en, ru string) {
	// Get an iterator for a new row at the end of the list store
	iter := listStore.Append()

	err := listStore.Set(iter,
		[]int{columnEN, columnRU},
		[]interface{}{en, ru})
	if err != nil {
		log.Fatal("Unable to add row:", err)
	}

}

func (win *TmplWindow) Run(tmpls []TTmpl) {
	//Выводим в таблицу
	for _, line := range tmpls {
		addRow(win.ListStore, line.En, line.Ru)
	}

	dialog = dialogWindowCreate()
	dialog.Window.SetTitle(win.locale.Sprintf("Replacement pattern"))
	dialog.TextEn.SetPlaceholderText(win.locale.Sprintf("Original text"))
	dialog.TextRu.SetPlaceholderText(win.locale.Sprintf("Text to replace"))

	dialog.BtnOk.Connect("clicked", func() {
		txtEn, err := dialog.TextEn.GetText()
		errorCheck(err)

		txtRu, err := dialog.TextRu.GetText()
		errorCheck(err)

		if dialog.NewItem {
			//log.Println("Добавляем новую строку")
			addRow(win.ListStore, txtEn, txtRu)
		} else {
			//log.Println("Изменяем строку")
			win.ListStore.SetValue(win.Iterator, columnEN, txtEn)
			win.ListStore.SetValue(win.Iterator, columnRU, txtRu)

		}

		dialog.Window.Hide()

	})

	win.Window.Show()

}

func (win *TmplWindow) GetTmpls() []TTmpl {

	var line TTmpl
	outdata := make([]TTmpl, 0)

	iter, _ := win.ListStore.GetIterFirst()
	next := true
	for next {
		valueEn, err := win.ListStore.GetValue(iter, columnEN)
		errorCheck(err)
		valueRu, err := win.ListStore.GetValue(iter, columnRU)
		errorCheck(err)

		line.En, _ = valueEn.GetString()
		line.Ru, _ = valueRu.GetString()

		outdata = append(outdata, line)

		next = win.ListStore.IterNext(iter)

	}

	return outdata
}

func (win *TmplWindow) lineSelected() {
	_, win.Iterator, _ = win.LineSelection.GetSelected()

	value, err := win.ListStore.GetValue(win.Iterator, columnEN)
	errorCheck(err)
	strEN, err := value.GetString()
	errorCheck(err)
	dialog.TextEn.SetText(strEN)

	value, err = win.ListStore.GetValue(win.Iterator, columnRU)
	errorCheck(err)
	strRU, err := value.GetString()
	errorCheck(err)
	dialog.TextRu.SetText(strRU)

	dialog.NewItem = false
	dialog.Window.Run()

}

// Применение выбранного языка
func (win *TmplWindow) SetLocale(locale *locales.Printer) {
	win.locale = locale
	win.Window.SetTitle(locale.Sprintf("Template editor"))
	win.Label_Head.SetText(locale.Sprintf("Templates for replacement") + "\n" + locale.Sprintf("The text is case sensitive"))
	win.BtnAdd.SetLabel(locale.Sprintf("Add"))
	win.BtnDel.SetLabel(locale.Sprintf("Del"))
}

func errorCheck(e error, text_opt ...string) {
	if e != nil {

		if len(text_opt) > 0 {
			log.Println(text_opt[0])
		}
		// panic for any errors.
		log.Panic(e)
	}
}
