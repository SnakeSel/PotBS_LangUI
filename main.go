// main.go
package main

import (
	"log"

	"snakesel/PotBS_LangUI/pkg/gtkutils"
	"snakesel/PotBS_LangUI/pkg/potbs"
	"snakesel/PotBS_LangUI/pkg/tmpl"

	"path/filepath"
	"sort"
	"strconv"
	str "strings"

	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/glib"
	"github.com/gotk3/gotk3/gtk"
)

var TmplList []tmpl.TTmpl

// IDs to access the tree view columns by
const (
	columnID = iota
	columnMode
	columnEN
	columnRU
	columnRuColor
)

const (
	test      = true
	appId     = "snakesel.potbs-langui"
	MainGlade = "data/main.glade"
	tmplFile  = "data/tmpl"
)

type Tlang struct {
	id   string
	mode string
	en   string
	ru   string
}

type MainWindow struct {
	Window *gtk.Window

	TreeView  *gtk.TreeView
	ListStore *gtk.ListStore

	LineSelection *gtk.TreeSelection

	BtnClose *gtk.Button
	BtnUp    *gtk.Button
	BtnDown  *gtk.Button

	Search *gtk.SearchEntry

	ToolBtnSave       *gtk.ToolButton
	ToolBtnSaveAs     *gtk.ToolButton
	ToolBtnTmpl       *gtk.ToolButton
	ToolSwitchDir     *gtk.ToggleToolButton
	ToolSwitchCopyBuf *gtk.ToggleToolButton

	Renderer_ru *gtk.CellRendererText

	FilePatch string
	FileName  string

	Iterator *gtk.TreeIter
}

type DialogWindow struct {
	Window *gtk.Dialog

	TextEn *gtk.TextView
	TextRu *gtk.TextView

	BufferEn *gtk.TextBuffer
	BufferRu *gtk.TextBuffer

	BtnCancel  *gtk.Button
	BtnOk      *gtk.Button
	BtnTmplRun *gtk.Button

	Label *gtk.Label
}

// Append a row to the list store for the tree view
func addRow(listStore *gtk.ListStore, id, tpe, en, ru string) {
	// Get an iterator for a new row at the end of the list store
	iter := listStore.Append()

	color := *gdk.NewRGBA(250, 50, 50, 1)
	//color.SetColors(250, 80, 80, 1)

	// color.SetColors(0.0, 0.0, 0.0, 0.0)
	// Set the contents of the list store row that the iterator represents
	err := listStore.Set(iter,
		[]int{columnID, columnMode, columnEN, columnRU, columnRuColor},
		[]interface{}{id, tpe, en, ru, color})
	if err != nil {
		log.Fatal("Unable to add row:", err)
	}

}

func main() {

	// Create a new application.
	application, err := gtk.ApplicationNew(appId, glib.APPLICATION_FLAGS_NONE)
	errorCheck(err)

	// Connect function to application activate event
	application.Connect("activate", func() {
		// Создаём билдер
		b, err := gtk.BuilderNewFromFile(MainGlade)
		errorCheck(err, "Error: No load main.glade")

		win := mainWindowCreate(b)
		dialog := dialogWindowCreate(b)

		// Map the handlers to callback functions, and connect the signals
		// to the Builder.
		signals := map[string]interface{}{
			"main_btn_save_clicked":       win.ToolBtnSave_clicked,
			"main_btn_saveas_clicked":     win.ToolBtnSaveAs_clicked,
			"main_btn_tmpl_clicked":       win.ToolBtnTmpl_clicked,
			"dialog_btn_tmpl_run_clicked": dialog.BtnTmplRun_clicked,
		}
		b.ConnectSignals(signals)

		// Сигналы MainWindow
		win.Window.Connect("destroy", func() {
			application.Quit()
		})

		win.BtnDown.Connect("clicked", func() {
			searchtext, _ := win.Search.GetText()
			patch := win.searchNext(searchtext)
			win.TreeView.SetCursor(patch, nil, false)
		})

		win.BtnUp.Connect("clicked", func() {
			searchtext, _ := win.Search.GetText()
			patch := win.searchPrev(searchtext)
			win.TreeView.SetCursor(patch, nil, false)
		})

		win.Search.Connect("search-changed", func() {
			searchtext, _ := win.Search.GetText()
			patch := win.searchNext(searchtext)
			win.TreeView.SetCursor(patch, nil, false)
		})

		win.BtnClose.Connect("clicked", func() {
			win.Window.Close()
		})

		win.TreeView.Connect("row-activated", func() {
			win.lineSelected(dialog)
		})

		//Сигналы dialog_translite
		dialog.BtnCancel.Connect("clicked", func() {
			dialog.Window.Hide()
		})

		dialog.BtnOk.Connect("clicked", func() {
			txt, err := dialog.BufferRu.GetText(dialog.BufferRu.GetStartIter(), dialog.BufferRu.GetEndIter(), true)
			errorCheck(err)

			win.ListStore.SetValue(win.Iterator, columnRU, txt)
			dialog.Window.Hide()
		})

		// #########################################
		// Загружаем файлы перевода
		win.loadFiles()

		// Загружаем шаблоны
		TmplList = tmpl.LoadTmplFromFile(tmplFile)

		// Отображаем все виджеты в окне
		win.Window.Show()

		application.AddWindow(win.Window)
		application.AddWindow(dialog.Window)

	})

	application.Run(nil)

}

func mainWindowCreate(b *gtk.Builder) *MainWindow {

	win := new(MainWindow)

	// Получаем объект главного окна по ID
	obj, err := b.GetObject("window_main")
	errorCheck(err, "Error: No find window_main")

	// Преобразуем из объекта именно окно типа gtk.Window
	// и соединяем с сигналом "destroy" чтобы можно было закрыть
	// приложение при закрытии окна
	win.Window = obj.(*gtk.Window)

	// Получаем остальные объекты window_main
	win.TreeView = gtkutils.GetTreeView(b, "treeview")
	win.ListStore = gtkutils.GetListStore(b, "liststore")
	win.LineSelection = gtkutils.GetTreeSelection(b, "LineSelection")
	win.Renderer_ru = gtkutils.GetCellRendererText(b, "renderer_ru")

	win.Search = gtkutils.GetSearchEntry(b, "entry_search")

	win.ToolBtnSave = gtkutils.GetToolButton(b, "tool_btn_save")
	win.ToolBtnSaveAs = gtkutils.GetToolButton(b, "tool_btn_saveAs")
	win.ToolBtnTmpl = gtkutils.GetToolButton(b, "tool_btn_tmpl")
	win.ToolSwitchDir = gtkutils.GetToggleToolButton(b, "tool_switch_dir")
	//win.ToolSwitchDir.SetActive(true)
	win.ToolSwitchCopyBuf = gtkutils.GetToggleToolButton(b, "tool_switch_copy_buf")

	win.BtnClose = gtkutils.GetButton(b, "button_close")
	win.BtnUp = gtkutils.GetButton(b, "btn_up")
	win.BtnDown = gtkutils.GetButton(b, "btn_down")

	return win
}

func dialogWindowCreate(b *gtk.Builder) *DialogWindow {
	// Окно диалога
	dialog := new(DialogWindow)

	obj, err := b.GetObject("dialog_translite")
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
	dialog.TextEn = gtkutils.GetTextView(b, "dialog_text_en")
	dialog.TextRu = gtkutils.GetTextView(b, "dialog_text_ru")

	dialog.BufferEn = gtkutils.GetTextBuffer(b, "dialog_buffer_en")
	dialog.BufferRu = gtkutils.GetTextBuffer(b, "dialog_buffer_ru")

	dialog.BtnCancel = gtkutils.GetButton(b, "dialog_btn_cancel")
	dialog.BtnOk = gtkutils.GetButton(b, "dialog_btn_ok")
	dialog.BtnTmplRun = gtkutils.GetButton(b, "dialog_btn_tmpl_run")

	dialog.Label = gtkutils.GetLabel(b, "dialog_label")

	return dialog
}

func (win *MainWindow) loadFiles() {

	var lang Tlang
	DataALL := make(map[string]Tlang)

	// Load EN
	if test {
		win.FilePatch = "/home/mks/Pirates of the Burning Sea/locale"
	}

	win.getFileName("Выберите исходный файл для перевода")

	Data := potbs.ReadDat(win.FileName)
	for _, line := range Data {
		lang.id = line.Id
		lang.mode = line.Mode
		lang.en = line.Text
		//DataALL[line.Id+line.Mode] = lang

		//test 2
		// Проверяем, если уже есть такой id, добавляем _ (т.к. id+mode не уникален)
		if _, ok := DataALL[line.Id+line.Mode]; ok {
			DataALL[line.Id+line.Mode+"_"] = lang
		} else {
			DataALL[line.Id+line.Mode] = lang
		}
	}
	log.Printf("%s успешно загружен", filepath.Base(win.FileName))

	// Load RU
	win.getFileName("Выберите Русский файл")

	Data = potbs.ReadDat(win.FileName)
	tmpmap := make(map[string]bool)
	for _, line := range Data {
		lang.id = line.Id
		lang.mode = line.Mode
		//lang.en = DataALL[line.Id+line.Mode].en
		lang.ru = line.Text
		//DataALL[line.Id+line.Mode] = lang

		//test 2
		// Проверяем, е5сли уже есть такой id, добавляем _ (т.к. id+mode не уникален)
		if _, ok := tmpmap[line.Id+line.Mode]; ok {
			lang.en = DataALL[line.Id+line.Mode+"_"].en
			DataALL[line.Id+line.Mode+"_"] = lang
			tmpmap[line.Id+line.Mode+"_"] = true
		} else {
			lang.en = DataALL[line.Id+line.Mode].en
			DataALL[line.Id+line.Mode] = lang
			tmpmap[line.Id+line.Mode] = true
		}
	}
	log.Printf("%s успешно загружен", filepath.Base(win.FileName))

	//Сортируем
	lines := make([]Tlang, 0, len(DataALL))

	for _, v := range DataALL {
		lines = append(lines, Tlang{v.id, v.mode, v.en, v.ru})
	}

	sort.SliceStable(lines, func(i, j int) bool {
		before, _ := strconv.Atoi(lines[i].id)
		next, _ := strconv.Atoi(lines[j].id)
		return before < next
	})

	//Выводим в таблицу
	for _, line := range lines {
		addRow(win.ListStore, line.id, line.mode, line.en, line.ru)
	}
	//color := *gdk.NewRGBA(239,41,41)
	//win.Renderer_ru.SetProperty("background-rgba", win.ListStore.GetColumnType(columnRuColor))

	//win.Renderer_ru.SetProperty("background-set", true)
}

func (win *MainWindow) ToolBtnSave_clicked() {
	dialog := gtk.MessageDialogNew(win.Window, gtk.DIALOG_MODAL, gtk.MESSAGE_INFO, gtk.BUTTONS_OK_CANCEL, "Внимание!")
	dialog.FormatSecondaryText("Вы уверены, что хотите перезаписать\n" + win.FileName + " ?")
	resp := dialog.Run()
	dialog.Close()
	if resp == gtk.RESPONSE_OK {
		savedatfile(win, win.FileName)
		//win.Window.Destroy()
	}

}

func (win *MainWindow) ToolBtnSaveAs_clicked() {
	native, err := gtk.FileChooserNativeDialogNew("Выберите файл для сохранения", win.Window, gtk.FILE_CHOOSER_ACTION_SAVE, "OK", "Cancel")
	errorCheck(err)
	native.SetCurrentFolder(win.FilePatch)
	native.SetCurrentName("out.dat")
	resp := native.Run()

	if resp == int(gtk.RESPONSE_ACCEPT) {
		log.Println(native.GetFilename())
		savedatfile(win, native.GetFilename())
		win.Window.Destroy()
	}

}
func (win *MainWindow) ToolBtnTmpl_clicked() {

	wintmpl := tmpl.TmplWindowCreate()

	wintmpl.BtnSave.Connect("clicked", func() {
		TmplList = wintmpl.GetTmpls()
		tmpl.SaveTmplToFile(TmplList, tmplFile)
		wintmpl.Window.Destroy()
	})

	//wintmpl.Window.SetParent(win.Window)
	wintmpl.Run(TmplList)
}

func (win *MainWindow) getFileName(title string) {

	filter_dat, err := gtk.FileFilterNew()
	errorCheck(err)
	filter_dat.AddPattern("*.dat")
	filter_dat.SetName(".dat")

	filter_all, err := gtk.FileFilterNew()
	errorCheck(err)
	filter_all.AddPattern("*")
	filter_all.SetName("Any files")

	native, err := gtk.FileChooserNativeDialogNew(title, win.Window, gtk.FILE_CHOOSER_ACTION_OPEN, "OK", "Cancel")
	errorCheck(err)

	if win.FilePatch != "" {
		native.SetCurrentFolder(win.FilePatch)
	}
	native.AddFilter(filter_dat)
	native.AddFilter(filter_all)
	native.SetFilter(filter_dat)

	respons := native.Run()

	// NativeDialog возвращает int с кодом ответа. -3 это GTK_RESPONSE_ACCEPT
	if respons != int(gtk.RESPONSE_ACCEPT) {
		win.Window.Close()
		log.Fatal("Отмена выбора файла")
	}
	win.FilePatch, _ = native.GetCurrentFolder()
	win.FileName = native.GetFilename()

	native.Destroy()
}

func savedatfile(win *MainWindow, outfile string) {

	var line potbs.TData
	outdata := make([]potbs.TData, 0)

	iter, _ := win.ListStore.GetIterFirst()
	next := true
	for next {
		valueId, err := win.ListStore.GetValue(iter, columnID)
		errorCheck(err)
		valueMode, err := win.ListStore.GetValue(iter, columnMode)
		errorCheck(err)
		valueRu, err := win.ListStore.GetValue(iter, columnRU)
		errorCheck(err)

		line.Id, _ = valueId.GetString()
		line.Mode, _ = valueMode.GetString()
		if line.Mode == "ucdt" {
			val, _ := valueRu.GetString()
			line.Text = str.ReplaceAll(val, "\t", " ")
		} else {
			line.Text, _ = valueRu.GetString()
		}

		// Если русского перевода нет, а в англиском текст есть, не записываем
		//if len(line.Text) == 0 {
		if line.Text == "" {
			valueEn, err := win.ListStore.GetValue(iter, columnEN)
			errorCheck(err)
			val, _ := valueEn.GetString()
			//if len(val) > 0 && line.Mode != "ucdn" {
			if val != "" && line.Mode != "ucdn" {
				next = win.ListStore.IterNext(iter)
				continue
			}
		}

		outdata = append(outdata, line)

		next = win.ListStore.IterNext(iter)

	}

	dirs := potbs.SaveDat(outfile, outdata)

	if win.ToolSwitchDir.GetActive() {
		patch, file := filepath.Split(outfile)
		potbs.SaveDir(patch+str.TrimSuffix(file, filepath.Ext(file))+".dir", dirs)
	}

}

func (win *MainWindow) lineSelected(dialog *DialogWindow) {
	_, win.Iterator, _ = win.LineSelection.GetSelected()

	value, err := win.ListStore.GetValue(win.Iterator, columnEN)
	errorCheck(err)
	strEN, err := value.GetString()
	errorCheck(err)
	dialog.BufferEn.SetText(strEN)

	value, err = win.ListStore.GetValue(win.Iterator, columnRU)
	errorCheck(err)
	strRU, err := value.GetString()
	errorCheck(err)
	dialog.BufferRu.SetText(strRU)

	value, err = win.ListStore.GetValue(win.Iterator, columnID)
	errorCheck(err)
	strID, err := value.GetString()
	errorCheck(err)
	dialog.Label.SetText(strID)

	if win.ToolSwitchCopyBuf.GetActive() {
		clip, _ := gtk.ClipboardGet(gdk.SELECTION_CLIPBOARD)
		clip.SetText(strID + "\t" + strEN + "\t")
	}

	dialog.Window.Run()

}

func (win *MainWindow) searchNext(text string) *gtk.TreePath {
	var loop int
	var next bool

	_, iter, ok := win.LineSelection.GetSelected()
	if !ok {
		iter, _ = win.ListStore.GetIterFirst()
	}

	searchtext := str.ToUpper(text)
	loop = 1
	for loop < 3 {
		next = win.ListStore.IterNext(iter)
		if !next {
			iter, _ = win.ListStore.GetIterFirst()
			loop += 1
		}
		valueId, err := win.ListStore.GetValue(iter, columnID)
		errorCheck(err)
		valueEn, err := win.ListStore.GetValue(iter, columnEN)
		errorCheck(err)
		valueRu, err := win.ListStore.GetValue(iter, columnRU)
		errorCheck(err)

		Id, _ := valueId.GetString()
		En, _ := valueEn.GetString()
		Ru, _ := valueRu.GetString()

		if str.Contains(str.ToUpper(Id), searchtext) || str.Contains(str.ToUpper(En), searchtext) || str.Contains(str.ToUpper(Ru), searchtext) {

			patch, err := win.ListStore.GetPath(iter)
			errorCheck(err)

			loop = 100

			return patch
		}

	}
	return nil
}

func (win *MainWindow) searchPrev(text string) *gtk.TreePath {
	var loop int
	var prev bool

	_, iter, ok := win.LineSelection.GetSelected()
	if !ok {
		iter, _ = win.ListStore.GetIterFirst()
	}

	searchtext := str.ToUpper(text)
	loop = 1
	for loop < 3 {
		prev = win.ListStore.IterPrevious(iter)
		if !prev {
			iter, _ = win.ListStore.GetIterFirst()
			loop += 1
		}
		valueId, err := win.ListStore.GetValue(iter, columnID)
		errorCheck(err)
		valueEn, err := win.ListStore.GetValue(iter, columnEN)
		errorCheck(err)
		valueRu, err := win.ListStore.GetValue(iter, columnRU)
		errorCheck(err)

		Id, _ := valueId.GetString()
		En, _ := valueEn.GetString()
		Ru, _ := valueRu.GetString()

		if str.Contains(str.ToUpper(Id), searchtext) || str.Contains(str.ToUpper(En), searchtext) || str.Contains(str.ToUpper(Ru), searchtext) {

			patch, err := win.ListStore.GetPath(iter)
			errorCheck(err)

			loop = 100

			return patch
		}

	}
	return nil
}

// Заменяем текст оригинала по шаблонам
func (dialog *DialogWindow) BtnTmplRun_clicked() {

	text, err := dialog.BufferEn.GetText(dialog.BufferEn.GetStartIter(), dialog.BufferEn.GetEndIter(), true)
	errorCheck(err)

	for _, line := range TmplList {
		text = str.ReplaceAll(text, line.En, line.Ru)
	}

	dialog.BufferRu.SetText(text)

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
