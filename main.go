// main.go

package main

import (
	"log"

	"snakesel/PotBS_LangUI/pkg/gtkutils"
	"snakesel/PotBS_LangUI/pkg/potbs"
	"snakesel/PotBS_LangUI/pkg/tmpl"

	tr "github.com/bas24/googletranslatefree"

	"path/filepath"
	"sort"
	"strconv"
	str "strings"

	//	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/glib"
	"github.com/gotk3/gotk3/gtk"

	//"os"

	"gopkg.in/ini.v1"
)

const (
	version   = "20200329"
	appId     = "snakesel.potbs-langui"
	MainGlade = "data/main.glade"
	tmplPatch = "data/tmpl"
	cfgFile   = "data/cfg.ini"
)

var TmplList []tmpl.TTmpl
var cfg *ini.File

// Временные переменные доступные во всех функциях
type tEnv struct {
	sourceLang         string        // исходный язык для перевода
	targetLang         string        // на какой будем переводить
	tmplFile           string        // Файл шаблонов для языка (tmplPatch_sourceLang-targetLang)
	filterChildEndIter *gtk.TreeIter // Хранит итератор последней записи. используется при обратном поиске
}

var env tEnv

// IDs to access the tree view columns by
const (
	columnID = iota
	columnMode
	columnEN
	columnRU
	//columnRuColor
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
	Filter    *gtk.TreeModelFilter

	LineSelection *gtk.TreeSelection

	BtnClose *gtk.Button
	BtnUp    *gtk.Button
	BtnDown  *gtk.Button

	Search      *gtk.SearchEntry
	Search_Full *gtk.CheckButton
	bnt_filter  *gtk.ToggleButton

	ToolBtnSave   *gtk.ToolButton
	ToolBtnSaveAs *gtk.ToolButton
	ToolBtnTmpl   *gtk.ToolButton

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
	BtnGooglTr *gtk.Button

	Label *gtk.Label
}

// Append a row to the list store for the tree view
func addRow(listStore *gtk.ListStore, id, tpe, en, ru string) error {
	// Get an iterator for a new row at the end of the list store
	iter := listStore.Append()

	//color := *gdk.NewRGBA(250, 50, 50, 1)
	//color.SetColors(250, 80, 80, 1)

	// color.SetColors(0.0, 0.0, 0.0, 0.0)
	// Set the contents of the list store row that the iterator represents
	err := listStore.Set(iter,
		[]int{columnID, columnMode, columnEN, columnRU},
		[]interface{}{id, tpe, en, ru})
	if err != nil {
		log.Fatal("Unable to add row:", err)
	}
	return err

}

func main() {
	var err error

	log.Printf("Запуск PotBS_LangUI, версия: %s\n", version)

	// Загрузка настроек
	cfg, err = ini.LooseLoad(cfgFile)
	if err != nil {
		log.Fatalf("Fail to read file: %v", err)
	}

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
			"main_btn_save_clicked":        win.ToolBtnSave_clicked,
			"main_btn_saveas_clicked":      win.ToolBtnSaveAs_clicked,
			"main_btn_tmpl_clicked":        win.ToolBtnTmpl_clicked,
			"main_btn_filter_clicked":      win.BtnFilter_clicked,
			"dialog_btn_tmpl_run_clicked":  dialog.BtnTmplRun_clicked,
			"dialog_btn_google_tr_clicked": dialog.BtnGoogleTr_clicked,
		}
		b.ConnectSignals(signals)

		// Сигналы MainWindow
		win.Window.Connect("destroy", func() {
			application.Quit()
		})
		//Сохранение настроек при закрытии окна
		win.Window.Connect("delete-event", func() {
			w, h := win.Window.GetSize()
			cfg.Section("Main").Key("width").SetValue(strconv.Itoa(w))
			cfg.Section("Main").Key("height").SetValue(strconv.Itoa(h))

			x, y := win.Window.GetPosition()
			cfg.Section("Main").Key("posX").SetValue(strconv.Itoa(x))
			cfg.Section("Main").Key("posY").SetValue(strconv.Itoa(y))

			cfg.Section("Main").Key("Patch").SetValue(win.FilePatch)

			w, h = dialog.Window.GetSize()
			cfg.Section("Translate").Key("width").SetValue(strconv.Itoa(w))
			cfg.Section("Translate").Key("height").SetValue(strconv.Itoa(h))

			//x, y = dialog.Window.GetPosition()
			//cfg.Section("Translate").Key("posX").SetValue(strconv.Itoa(x))
			//cfg.Section("Translate").Key("posY").SetValue(strconv.Itoa(y))

			cfg.SaveTo(cfgFile)
		})

		win.BtnDown.Connect("clicked", func() {
			searchtext, _ := win.Search.GetText()
			patch := win.searchNext(searchtext)
			if patch != nil {
				win.TreeView.SetCursor(patch, nil, false)
			}
		})

		win.BtnUp.Connect("clicked", func() {
			searchtext, _ := win.Search.GetText()
			patch := win.searchPrev(searchtext)
			if patch != nil {
				win.TreeView.SetCursor(patch, nil, false)
			}
		})

		win.Search.Connect("search-changed", func() {
			searchtext, _ := win.Search.GetText()
			patch := win.searchNext(searchtext)
			if patch != nil {
				win.TreeView.SetCursor(patch, nil, false)
			}
		})

		win.BtnClose.Connect("clicked", func() {
			win.Window.Close()
		})

		win.TreeView.Connect("row-activated", func() {
			win.lineSelected(dialog)
		})

		//Сигналы dialog_translate
		dialog.BtnCancel.Connect("clicked", func() {
			dialog.Window.Hide()
		})

		dialog.BtnOk.Connect("clicked", func() {
			txt, err := dialog.BufferRu.GetText(dialog.BufferRu.GetStartIter(), dialog.BufferRu.GetEndIter(), true)
			errorCheck(err)

			win.ListStore.SetValue(win.Filter.ConvertIterToChildIter(win.Iterator), columnRU, txt)
			dialog.Window.Hide()
		})

		// ### применяем настроки
		win.Window.Resize(cfg.Section("Main").Key("width").MustInt(600), cfg.Section("Main").Key("height").MustInt(600))
		win.Window.Move(cfg.Section("Main").Key("posX").MustInt(0), cfg.Section("Main").Key("posY").MustInt(0))

		dialog.Window.Resize(cfg.Section("Translate").Key("width").MustInt(900), cfg.Section("Translate").Key("height").MustInt(300))
		//dialog.Window.Move(cfg.Section("Translate").Key("posX").MustInt(0), cfg.Section("Translate").Key("posY").MustInt(0))

		// #########################################
		// Загружаем файлы перевода
		win.loadFiles()

		env.tmplFile = tmplPatch + "_" + env.sourceLang + "-" + env.targetLang
		// Загружаем шаблоны
		TmplList = tmpl.LoadTmplFromFile(env.tmplFile)

		win.Filter.SetVisibleFunc(win.Filter_Clear)
		win.Filter.Refilter()
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
	win.Filter = gtkutils.GetTreeModelFilter(b, "treeFilter")
	win.Renderer_ru = gtkutils.GetCellRendererText(b, "renderer_ru")

	win.Search = gtkutils.GetSearchEntry(b, "entry_search")
	win.Search_Full = gtkutils.GetCheckButton(b, "chk_full")
	win.bnt_filter = gtkutils.GetToggleButton(b, "bnt_filter")

	win.ToolBtnSave = gtkutils.GetToolButton(b, "tool_btn_save")
	win.ToolBtnSaveAs = gtkutils.GetToolButton(b, "tool_btn_saveAs")
	win.ToolBtnTmpl = gtkutils.GetToolButton(b, "tool_btn_tmpl")

	win.BtnClose = gtkutils.GetButton(b, "button_close")
	win.BtnUp = gtkutils.GetButton(b, "btn_up")
	win.BtnDown = gtkutils.GetButton(b, "btn_down")

	return win
}

// Окно диалога
func dialogWindowCreate(b *gtk.Builder) *DialogWindow {

	dialog := new(DialogWindow)

	obj, err := b.GetObject("dialog_translite")
	errorCheck(err)
	dialog.Window = obj.(*gtk.Dialog)
	dialog.Window.Connect("close", func() {
		dialog.Window.Hide()
	})
	// dialog.Window.Connect("delete-event", func() {
	// 	dialog.Window.Hide()
	// })

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
	dialog.BtnGooglTr = gtkutils.GetButton(b, "dialog_btn_googletr")

	dialog.Label = gtkutils.GetLabel(b, "dialog_label")

	return dialog
}

func (win *MainWindow) loadFiles() {

	var lang Tlang
	DataALL := make(map[string]Tlang)

	// Load source Lang
	win.FilePatch = cfg.Section("Main").Key("Patch").MustString("")

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
	env.sourceLang = str.ToUpper(filepath.Base(win.FileName)[0:2]) //Добавить проверки

	log.Printf("%s успешно загружен", filepath.Base(win.FileName))

	// Load target Lang
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
	env.targetLang = str.ToUpper(filepath.Base(win.FileName)[0:2]) //Добавить проверки

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
		err := addRow(win.ListStore, line.id, line.mode, line.en, line.ru)
		errorCheck(err)
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
	native.SetCurrentFolder(cfg.Section("Main").Key("Patch").MustString(""))
	native.SetCurrentName("out.dat")
	resp := native.Run()

	if resp == int(gtk.RESPONSE_ACCEPT) {
		log.Println(native.GetFilename())
		savedatfile(win, native.GetFilename())
		//win.Window.Destroy()
	}

}

func (win *MainWindow) Filter_Clear(model *gtk.TreeModelFilter, iter *gtk.TreeIter, userData ...interface{}) bool {

	if !win.bnt_filter.GetActive() {
		env.filterChildEndIter = iter
		return true
	}

	value, _ := model.GetValue(iter, columnRU)
	textRU, _ := value.GetString()

	value, _ = model.GetValue(iter, columnEN)
	textEN, _ := value.GetString()

	if (textRU == "") && (textEN != "") {
		env.filterChildEndIter = iter
		return true
	} else {
		return false
	}

}

func (win *MainWindow) BtnFilter_clicked() {
	win.Filter.Refilter()
}

func (win *MainWindow) ToolBtnTmpl_clicked() {

	wintmpl := tmpl.TmplWindowCreate()
	wintmpl.Col_SourceLang.SetTitle(env.sourceLang)
	wintmpl.Col_TargetLang.SetTitle(env.targetLang)

	wintmpl.Window.Resize(cfg.Section("Template").Key("width").MustInt(900), cfg.Section("Template").Key("height").MustInt(400))
	wintmpl.Window.Move(cfg.Section("Template").Key("posX").MustInt(0), cfg.Section("Template").Key("posY").MustInt(0))

	wintmpl.BtnSave.Connect("clicked", func() {
		TmplList = wintmpl.GetTmpls()

		// Сортируем от ольшего совпадения к меньшему
		sort.SliceStable(TmplList, func(i, j int) bool {
			before := len(TmplList[i].En)
			next := len(TmplList[j].En)
			return before > next
		})

		tmpl.SaveTmplToFile(TmplList, env.tmplFile)

		wintmpl.Window.Close()
	})
	//Сохранение настроек при закрытии окна
	wintmpl.Window.Connect("delete-event", func() {
		w, h := wintmpl.Window.GetSize()
		cfg.Section("Template").Key("width").SetValue(strconv.Itoa(w))
		cfg.Section("Template").Key("height").SetValue(strconv.Itoa(h))

		x, y := wintmpl.Window.GetPosition()
		cfg.Section("Template").Key("posX").SetValue(strconv.Itoa(x))
		cfg.Section("Template").Key("posY").SetValue(strconv.Itoa(y))
		cfg.SaveTo(cfgFile)
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

// Сохраняем перевод
func savedatfile(win *MainWindow, outfile string) {
	var sum_all, sum_ru int //Подсчет % перевода
	sum_all = 0
	sum_ru = 0

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

		//Подсчет % перевода
		if line.Mode != "ucdn" {
			sum_all += 1
			if line.Text != "" {
				sum_ru += 1
			}
		}

		// // Если русского перевода нет, а в англиском текст есть, не записываем
		// //if len(line.Text) == 0 {
		// if line.Text == "" {
		// 	valueEn, err := win.ListStore.GetValue(iter, columnEN)
		// 	errorCheck(err)
		// 	val, _ := valueEn.GetString()
		// 	//if len(val) > 0 && line.Mode != "ucdn" {
		// 	if val != "" && line.Mode != "ucdn" {
		// 		next = win.ListStore.IterNext(iter)
		// 		continue
		// 	}
		// }

		// Если русского перевода нет, и это текстовая строка (ucdt), пропускаем
		if line.Text == "" && line.Mode == "ucdt" {
			next = win.ListStore.IterNext(iter)
			continue
		}

		outdata = append(outdata, line)

		next = win.ListStore.IterNext(iter)

	}

	dirs := potbs.SaveDat(outfile, outdata)

	// Создаем dir файл
	patch, file := filepath.Split(outfile)
	potbs.SaveDir(patch+str.TrimSuffix(file, filepath.Ext(file))+".dir", dirs)

	log.Printf("Переведено %d из %d (%d%s)", sum_ru, sum_all, int((sum_ru*100)/sum_all), "%")
}

// Заполнение окна с переводом при клике на строку
func (win *MainWindow) lineSelected(dialog *DialogWindow) {
	_, win.Iterator, _ = win.LineSelection.GetSelected()

	//value, err := win.ListStore.GetValue(win.Iterator, columnEN)
	value, err := win.Filter.GetValue(win.Iterator, columnEN)
	errorCheck(err)
	strEN, err := value.GetString()
	errorCheck(err)
	dialog.BufferEn.SetText(strEN)

	//value, err = win.ListStore.GetValue(win.Iterator, columnRU)
	value, err = win.Filter.GetValue(win.Iterator, columnRU)
	errorCheck(err)
	strRU, err := value.GetString()
	errorCheck(err)
	dialog.BufferRu.SetText(strRU)

	//value, err = win.ListStore.GetValue(win.Iterator, columnID)
	value, err = win.Filter.GetValue(win.Iterator, columnID)
	errorCheck(err)
	strID, err := value.GetString()
	errorCheck(err)
	dialog.Label.SetText(strID)

	//dialog.Window.Run()
	dialog.Window.Show()

}

// Прямой поиск
func (win *MainWindow) searchNext(text string) *gtk.TreePath {

	var loop int

	_, iter, ok := win.LineSelection.GetSelected()
	if !ok {
		//iter, _ = win.ListStore.GetIterFirst()
		iter, _ = win.Filter.GetIterFirst()
	}

	searchtext := str.ToUpper(text)
	loop = 1
	for loop < 3 {

		// Берем следующую строку, если ее нет, значит дошли до конца - переходим к первой
		//if !win.ListStore.IterNext(iter) {
		if !win.Filter.IterNext(iter) {
			iter, _ = win.Filter.GetIterFirst()
			loop += 1
		}

		if !win.ListStore.IterIsValid(win.Filter.ConvertIterToChildIter(iter)) {
			log.Println("Warn: неверный итератор Next")
			continue
		}

		valueId, err := win.Filter.GetValue(iter, columnID)
		errorCheck(err)
		valueEn, err := win.Filter.GetValue(iter, columnEN)
		errorCheck(err)
		valueRu, err := win.Filter.GetValue(iter, columnRU)
		errorCheck(err)

		Id, _ := valueId.GetString()
		En, _ := valueEn.GetString()
		Ru, _ := valueRu.GetString()

		if win.Search_Full.GetActive() {
			if str.ToUpper(Id) == searchtext || str.ToUpper(En) == searchtext || str.ToUpper(Ru) == searchtext {

				patch, err := win.Filter.GetPath(iter)
				errorCheck(err)

				loop = 100

				return patch
			}

		} else {
			if str.Contains(str.ToUpper(Id), searchtext) || str.Contains(str.ToUpper(En), searchtext) || str.Contains(str.ToUpper(Ru), searchtext) {

				patch, err := win.Filter.GetPath(iter)
				errorCheck(err)

				loop = 100

				return patch
			}

		}
	}
	log.Printf("Поиск '%s': ничего не найдено.\n", searchtext)
	return nil
}

// Обратный поиск
func (win *MainWindow) searchPrev(text string) *gtk.TreePath {

	_, iter, ok := win.LineSelection.GetSelected()
	if !ok {
		//Iter = &win.EndIterator
		iter, _ = win.Filter.ConvertChildIterToIter(env.filterChildEndIter)
	}

	searchtext := str.ToUpper(text)
	loop := 1
	for loop < 3 {
		// Берем предыдущую строку, если ее нет, значит дошли до начала - переходим к последнему итератору
		if !win.Filter.IterPrevious(iter) {
			//*Iter = win.EndIterator
			iter, _ = win.Filter.ConvertChildIterToIter(env.filterChildEndIter)
			loop += 1
		}

		if !win.ListStore.IterIsValid(win.Filter.ConvertIterToChildIter(iter)) {
			log.Println("Warn: неверный итератор Prev")
			continue
		}

		// Получаем значения полей
		valueId, err := win.Filter.GetValue(iter, columnID)
		errorCheck(err)
		valueEn, err := win.Filter.GetValue(iter, columnEN)
		errorCheck(err)
		valueRu, err := win.Filter.GetValue(iter, columnRU)
		errorCheck(err)

		Id, _ := valueId.GetString()
		En, _ := valueEn.GetString()
		Ru, _ := valueRu.GetString()

		// Сравниваем значения полей с поисковой фразой
		if win.Search_Full.GetActive() {
			// Полное сравнение
			if str.ToUpper(Id) == searchtext || str.ToUpper(En) == searchtext || str.ToUpper(Ru) == searchtext {

				patch, err := win.Filter.GetPath(iter)
				errorCheck(err)

				loop = 100

				return patch
			}
		} else {
			// Сравнение по совпадению
			if str.Contains(str.ToUpper(Id), searchtext) || str.Contains(str.ToUpper(En), searchtext) || str.Contains(str.ToUpper(Ru), searchtext) {

				patch, err := win.Filter.GetPath(iter)
				errorCheck(err)

				loop = 100

				return patch
			}
		}

	}
	log.Printf("Поиск '%s': ничего не найдено.\n", searchtext)
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

// Переводим текст через Google Translate
func (dialog *DialogWindow) BtnGoogleTr_clicked() {

	text, err := dialog.BufferEn.GetText(dialog.BufferEn.GetStartIter(), dialog.BufferEn.GetEndIter(), true)
	errorCheck(err)

	// Заменяем текст оригинала по шаблонам. Для более точного перевода
	for _, line := range TmplList {
		text = str.ReplaceAll(text, line.En, line.Ru)
	}

	// отправляем в гугл
	res, err := tr.Translate(text, env.sourceLang, env.targetLang)
	errorCheck(err)

	dialog.BufferRu.SetText(res)

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
