// main.go

package main

import (
	"log"

	//"github.com/snakesel/potbs_langui/pkg/apkstrings"
	"github.com/snakesel/potbs_langui/pkg/gtkutils"
	"github.com/snakesel/potbs_langui/pkg/potbs"
	"github.com/snakesel/potbs_langui/pkg/tmpl"

	tr "github.com/bas24/googletranslatefree"

	"container/list"
	"path/filepath"
	"sort"
	"strconv"
	str "strings"

	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/glib"
	"github.com/gotk3/gotk3/gtk"

	"os"

	"github.com/tealeg/xlsx"
	"gopkg.in/ini.v1"
)

const (
	version   = "20201030"
	appId     = "snakesel.potbs-langui"
	MainGlade = "data/main.glade"
	tmplPatch = "data/tmpl"
	cfgFile   = "data/cfg.ini"
)

var TmplList []tmpl.TTmpl
var cfg *ini.File

type intProject interface {
	LoadFile(string) (*list.List, error)
	SaveFile(string, *list.List) error

	SetSourceLang(string)
	GetSourceLang() string
	SetTargetLang(string)
	GetTargetLang() string

	GetHeaderLen() int
	GetHeader() map[string]int
	GetHeaderNbyName(string) int
}

// IDs to access the tree view columns by
const (
	columnID = iota
	columnMode
	columnEN
	columnRU
	//columnRuColor
)

// ID фильтров
const (
	filterALL = iota
	filterNotTranslate
	filterNotOriginal
	filterUserFilter
)

type Tlang struct {
	id     string `xlsx:"0"`
	mode   string `xlsx:"1"`
	source string `xlsx:"2"`
	target string `xlsx:"3"`
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

	Search       *gtk.SearchEntry
	Search_Full  *gtk.CheckButton
	combo_filter *gtk.ComboBoxText
	userFilter   *gtk.Entry

	ToolBtnSave       *gtk.ToolButton
	ToolBtnSaveAs     *gtk.ToolButton
	ToolBtnTmpl       *gtk.ToolButton
	ToolBtnExportXLSX *gtk.ToolButton
	ToolBtnImportXLSX *gtk.ToolButton

	Renderer_ru *gtk.CellRendererText

	Iterator *gtk.TreeIter
	Project  intProject
	//Project *potbs.Translate

	tmplFile           string        // Файл шаблонов для языка (tmplPatch_sourceLang-targetLang)
	filterChildEndIter *gtk.TreeIter // Хранит итератор последней записи. используется при обратном поиске
	clearNotOriginal   bool          // не сохранять строки которых нет в оригинале
	//langFileFullPath   string        // Хранит путь к файлу и имя файла с переводом
	langFilePath string // Хранит путь к файлу с переводом
	langFileName string // Хранит только имя файла перевода
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

	Label      *gtk.Label
	sourceLang string // исходный язык для перевода
	targetLang string // на какой будем переводить
}

// Append a row to the list store for the tree view
func addRow(listStore *gtk.ListStore, id, tpe, en, ru string) error {
	// Get an iterator for a new row at the end of the list store
	iter := listStore.Append()

	// Set the contents of the list store row that the iterator represents
	err := listStore.Set(iter,
		[]int{columnID, columnMode, columnEN, columnRU},
		[]interface{}{id, tpe, en, ru})
	if err != nil {
		log.Fatal("[ERR]\tUnable to add row:", err)
	}
	return err

}

func main() {
	var err error

	// Загрузка настроек
	cfg, err = ini.LooseLoad(cfgFile)
	if err != nil {
		log.Fatalf("[ERR]\tFail to read file: %v", err)
	}

	// Если есть параметр, используем файл лога
	// Весь изврат из-за отсутствия вывода в консоль в Windows
	if file := cfg.Section("Main").Key("Log").MustString(""); file != "" {
		f, err := os.OpenFile(file, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
		if err != nil {
			log.Fatalf("[ERR]\terror opening file: %v", err)
		}
		defer f.Close()

		log.SetOutput(f)
	} else {
		log.SetOutput(os.Stdout)
	}

	log.Printf("Запуск PotBS_LangUI, версия: %s\n", version)

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
			"main_delete-event":            win.saveCfg, //Сохранение настроек при закрытии окна
			"main_btn_save_clicked":        win.ToolBtnSave_clicked,
			"main_btn_saveas_clicked":      win.ToolBtnSaveAs_clicked,
			"main_btn_export_xlsx_clicked": win.ToolBtnExportXLSX_clicked,
			"main_btn_import_xlsx_clicked": win.ToolBtnImportXLSX_clicked,
			"main_btn_tmpl_clicked":        win.ToolBtnTmpl_clicked,
			"main_combo_filter_change":     win.ComboFilter_clicked,
			"userfilter_activate":          win.ComboFilter_clicked,
			"dialog_btn_tmpl_run_clicked":  dialog.BtnTmplRun_clicked,
			"dialog_btn_google_tr_clicked": dialog.BtnGoogleTr_clicked,
		}
		b.ConnectSignals(signals)

		// Сигналы MainWindow
		win.Window.Connect("destroy", func() {
			dialog.saveCfg()
			application.Quit()
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

		win.Search.Connect("activate", func() {
			//win.Search.Connect("search-changed", func() {
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

		win.clearNotOriginal = cfg.Section("Main").Key("ClearNotOriginal").MustBool(false)

		// Путь к файлам
		win.langFilePath = cfg.Section("Main").Key("Patch").MustString("")

		// #########################################

		// Получаем пути к файлам перевода
		sourceFile, targetFile, extFile := win.getFileNames()

		switch extFile {
		case ".xls":
			//win.Project = apkstrings.New(apkstrings.Config{})
		default:
			// Проект перевода
			win.Project = potbs.New(potbs.Config{
				//Debug:     os.Stdout,
				//Debug: log.Writer(),
			})

		}

		// Загружаем язык перевода
		win.Project.SetSourceLang(cfg.Section("Project").Key("SourceLang").MustString(""))
		win.Project.SetTargetLang(cfg.Section("Project").Key("TargetLang").MustString(""))

		if extFile == ".dat" {
			// Если SourceLang не задан в настройках, то берем из имени файла
			// иначе проверяем из настроек
			if win.Project.GetSourceLang() == "" {
				win.Project.SetSourceLang(str.ToUpper(filepath.Base(sourceFile)[0:2]))
			}
			// Если даже теперь язык пуст, хреново
			if win.Project.GetSourceLang() == "" {
				log.Println("Не определить язык исходного файла")
			}

			// Если SourceLang не задан в настройках, то берем из имени файла
			// иначе проверяем из настроек
			if win.Project.GetTargetLang() == "" {
				win.Project.SetTargetLang(str.ToUpper(filepath.Base(targetFile)[0:2]))
			}
			// Если даже теперь язык пуст, хреново
			if win.Project.GetTargetLang() == "" {
				log.Println("Не определить язык конечного файла")
			}
		}

		// Загружаем файлы перевода и выводим в таблицу
		win.loadListStore(sourceFile, targetFile)

		win.TreeView.GetColumn(columnEN).SetTitle(win.Project.GetSourceLang())
		win.TreeView.GetColumn(columnRU).SetTitle(win.Project.GetTargetLang())

		dialog.sourceLang = win.Project.GetSourceLang()
		dialog.targetLang = win.Project.GetTargetLang()

		win.tmplFile = tmplPatch + "_" + win.Project.GetSourceLang() + "-" + win.Project.GetTargetLang()

		// Загружаем шаблоны
		TmplList = tmpl.LoadTmplFromFile(win.tmplFile)

		win.Filter.SetVisibleFunc(win.funcFilter)
		win.Filter.Refilter()

		// Отображаем все виджеты в окне
		win.Window.Show()

		dialog.Window.SetTransientFor(win.Window)

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
	win.combo_filter = gtkutils.GetComboBoxText(b, "combo_filter")
	win.userFilter = gtkutils.GetEntry(b, "entry_userfilter")

	win.ToolBtnSave = gtkutils.GetToolButton(b, "tool_btn_save")
	win.ToolBtnSaveAs = gtkutils.GetToolButton(b, "tool_btn_saveAs")
	win.ToolBtnTmpl = gtkutils.GetToolButton(b, "tool_btn_tmpl")
	win.ToolBtnExportXLSX = gtkutils.GetToolButton(b, "tool_btn_export_xlsx")
	win.ToolBtnImportXLSX = gtkutils.GetToolButton(b, "tool_btn_import_xlsx")

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

	// Перехват сигнала нажатия клавишь
	dialog.Window.Connect("key-press-event", dialog.keyPress, nil)

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

// Обработка нажатия клавишь в окне диалога
func (dialog *DialogWindow) keyPress(dial *gtk.Dialog, event *gdk.Event) bool {
	key := gdk.EventKeyNewFromEvent(event)

	if key.KeyVal() == gdk.KEY_Escape {
		dialog.Window.Hide()
		// true означает, что сигнал обработан
		// и далее его не надо передавать на стандартный обработчик
		return true

	}
	return false
}

// Запускает диалог выбора файла перевода
// с указанным заголовком
// возвращает полный путь к файлу
func (win *MainWindow) fileChooserFullPath(title string) string {

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

	if win.langFilePath != "" {
		native.SetCurrentFolder(win.langFilePath)
	}
	native.AddFilter(filter_dat)
	native.AddFilter(filter_all)
	native.SetFilter(filter_dat)

	respons := native.Run()

	// NativeDialog возвращает int с кодом ответа. -3 это GTK_RESPONSE_ACCEPT
	if respons != int(gtk.RESPONSE_ACCEPT) {
		win.Window.Close()
		log.Fatal("[INFO]\tОтмена выбора файла")
	}
	//win.langFilePath, _ = native.GetCurrentFolder()
	//win.langFileName = native.GetFilename()
	FileFullPath := native.GetFilename()

	native.Destroy()

	return FileFullPath
}

func (win *MainWindow) getFileNames() (sourceName, targetName, extName string) {

	// open source file
	sourceName = win.fileChooserFullPath("Выберите исходный файл для перевода (Select the source file to translate).")
	//win.langFilePath = filepath.Dir(sourceName)
	//win.langFileName = filepath.Base(sourceName)
	extName = filepath.Ext(sourceName)
	//log.Println(extName)

	// open target file
	targetName = win.fileChooserFullPath("Выберите файл перевода (Select the target file to translate)")
	win.langFilePath = filepath.Dir(targetName)
	win.langFileName = filepath.Base(targetName)

	return sourceName, targetName, extName
}

func (win *MainWindow) loadListStore(sourceName, targetName string) {

	id := win.Project.GetHeaderNbyName("id")
	mode := win.Project.GetHeaderNbyName("mode")
	text := win.Project.GetHeaderNbyName("text")

	var lang Tlang
	DataALL := make(map[string]Tlang)

	Data, err := win.Project.LoadFile(sourceName)
	errorCheck(err)

	for e := Data.Front(); e != nil; e = e.Next() {
		line := e.Value.([]string)

		lang.id = line[id]
		lang.mode = line[mode]
		lang.source = line[text]
		//DataALL[line.Id+line.Mode] = lang

		// Проверяем, если уже есть такой id, добавляем _ (т.к. id+mode не уникален)
		if _, ok := DataALL[line[id]+line[mode]]; ok {
			DataALL[line[id]+line[mode]+"_"] = lang
		} else {
			DataALL[line[id]+line[mode]] = lang
		}
	}

	log.Printf("[INFO]\t%s успешно загружен, язык: %s", filepath.Base(sourceName), win.Project.GetSourceLang())

	// Load target file

	Data, err = win.Project.LoadFile(targetName)
	errorCheck(err)

	tmpmap := make(map[string]bool)
	for e := Data.Front(); e != nil; e = e.Next() {
		line := e.Value.([]string)

		lang.id = line[id]
		lang.mode = line[mode]
		lang.target = line[text]
		//DataALL[line.Id+line.Mode] = lang

		// Проверяем, если уже есть такой id, добавляем _ (т.к. id+mode не уникален)
		if _, ok := tmpmap[line[id]+line[mode]]; ok {
			lang.source = DataALL[line[id]+line[mode]+"_"].source
			DataALL[line[id]+line[mode]+"_"] = lang
			tmpmap[line[id]+line[mode]+"_"] = true
		} else {
			lang.source = DataALL[line[id]+line[mode]].source
			DataALL[line[id]+line[mode]] = lang
			tmpmap[line[id]+line[mode]] = true
		}
	}

	log.Printf("[INFO]\t%s успешно загружен, язык: %s", filepath.Base(targetName), win.Project.GetTargetLang())

	//Сортируем
	lines := make([]Tlang, 0, len(DataALL))

	for _, v := range DataALL {
		lines = append(lines, Tlang{v.id, v.mode, v.source, v.target})
	}

	sort.SliceStable(lines, func(i, j int) bool {
		before, _ := strconv.Atoi(lines[i].id)
		next, _ := strconv.Atoi(lines[j].id)
		return before < next
	})

	//Выводим в таблицу
	for _, line := range lines {
		err := addRow(win.ListStore, line.id, line.mode, line.source, line.target)
		errorCheck(err)
	}
}

func (win *MainWindow) saveCfg() {
	//Сохранение настроек
	w, h := win.Window.GetSize()
	cfg.Section("Main").Key("width").SetValue(strconv.Itoa(w))
	cfg.Section("Main").Key("height").SetValue(strconv.Itoa(h))

	x, y := win.Window.GetPosition()
	cfg.Section("Main").Key("posX").SetValue(strconv.Itoa(x))
	cfg.Section("Main").Key("posY").SetValue(strconv.Itoa(y))

	cfg.Section("Main").Key("Patch").SetValue(win.langFilePath)

	cfg.SaveTo(cfgFile)
}

func (win *MainWindow) ToolBtnSave_clicked() {
	dialog := gtk.MessageDialogNew(win.Window, gtk.DIALOG_MODAL, gtk.MESSAGE_INFO, gtk.BUTTONS_OK_CANCEL, "Внимание!")
	dialog.FormatSecondaryText("Are you sure you want to overwrite:\nВы уверены, что хотите перезаписать:\n\n" + win.langFileName + " ?")
	resp := dialog.Run()
	dialog.Close()
	if resp == gtk.RESPONSE_OK {
		win.SaveTarget(filepath.Join(win.langFilePath, win.langFileName))
		//win.Window.Destroy()
	}

}

func (win *MainWindow) ToolBtnSaveAs_clicked() {
	native, err := gtk.FileChooserNativeDialogNew("Select a file to save\nВыберите файл для сохранения", win.Window, gtk.FILE_CHOOSER_ACTION_SAVE, "OK", "Cancel")
	errorCheck(err)
	native.SetCurrentFolder(win.langFilePath)
	native.SetCurrentName(win.Project.GetTargetLang() + "_data_mod.dat")
	resp := native.Run()

	if resp == int(gtk.RESPONSE_ACCEPT) {
		win.SaveTarget(native.GetFilename())
		//log.Printf("[INFO]\tФайл %s сохранен.\n", native.GetFilename())
		//win.Window.Destroy()
	}

}

func (win *MainWindow) ToolBtnExportXLSX_clicked() {
	native, err := gtk.FileChooserNativeDialogNew("Select a file to save\nВыберите файл для сохранения", win.Window, gtk.FILE_CHOOSER_ACTION_SAVE, "OK", "Cancel")
	errorCheck(err)
	native.SetCurrentFolder(cfg.Section("Main").Key("Patch").MustString(""))
	native.SetCurrentName(win.Project.GetSourceLang() + "-" + win.Project.GetTargetLang() + ".xlsx")
	resp := native.Run()

	if resp == int(gtk.RESPONSE_ACCEPT) {
		saveXLSXfile(win, native.GetFilename())
		log.Printf("[INFO]\tФайл %s сохранен.\n", native.GetFilename())
		//win.Window.Destroy()
	}

}

func (win *MainWindow) ToolBtnImportXLSX_clicked() {
	filter_dat, err := gtk.FileFilterNew()
	errorCheck(err)
	filter_dat.AddPattern("*.xlsx")
	filter_dat.SetName(".xlsx")

	filter_all, err := gtk.FileFilterNew()
	errorCheck(err)
	filter_all.AddPattern("*")
	filter_all.SetName("Any files")

	native, err := gtk.FileChooserNativeDialogNew("Select the XLSX file to import\nВыберите XLSX файл для импорта", win.Window, gtk.FILE_CHOOSER_ACTION_OPEN, "OK", "Cancel")
	errorCheck(err)

	native.SetCurrentFolder(win.langFilePath)

	native.AddFilter(filter_dat)
	native.AddFilter(filter_all)
	native.SetFilter(filter_dat)

	respons := native.Run()
	xlsfile := native.GetFilename()
	native.Destroy()
	// NativeDialog возвращает int с кодом ответа. -3 это GTK_RESPONSE_ACCEPT
	if respons != int(gtk.RESPONSE_ACCEPT) {
		return
	}

	dlg, _ := gtk.DialogNew()
	//dlg.SetParentWindow(win.Window)
	dlg.SetTitle("Import " + filepath.Base(xlsfile))
	dlg.AddButton("Не перевед. (untrans)", gtk.RESPONSE_ACCEPT)
	dlg.AddButton("Все (All)", gtk.RESPONSE_OK)
	dlg.AddButton("Отмена (Cancel)", gtk.RESPONSE_CANCEL)
	dlg.SetPosition(gtk.WIN_POS_CENTER)

	dlgBox, _ := dlg.GetContentArea()
	dlgBox.SetSpacing(6)

	lbl, _ := gtk.LabelNew("Импорт из первого листа в книге!\nЗаменить только не переведенные строки или все?\n\nImport from the first sheet in a book!\nChange only untranslated strings or all?")
	lbl.SetMarginStart(6)
	lbl.SetMarginEnd(6)
	//lbl.SetLineWrap(true)
	dlgBox.Add(lbl)
	lbl.Show()

	resp := dlg.Run()
	dlg.Destroy()

	switch resp {
	case gtk.RESPONSE_CANCEL:
		return
	case gtk.RESPONSE_ACCEPT:
		log.Println("[INFO]\tимпортируем только не переведенные из " + xlsfile)
		loadXLSXfile(win, xlsfile, false)

	case gtk.RESPONSE_OK:
		log.Println("[INFO]\tимпортируем все из " + xlsfile)
		loadXLSXfile(win, xlsfile, true)
	}

}

// Фильтр
func (win *MainWindow) funcFilter(model *gtk.TreeModelFilter, iter *gtk.TreeIter, userData ...interface{}) bool {

	switch win.combo_filter.GetActive() {
	case filterALL:
		// Фильтр всех записей
		if win.userFilter.GetVisible() {
			win.userFilter.SetVisible(false)
		}
		win.filterChildEndIter = iter
		return true
	case filterNotTranslate:
		// Фильтр Не переведенных записей
		if win.userFilter.GetVisible() {
			win.userFilter.SetVisible(false)
		}

		value, _ := model.GetValue(iter, columnRU)
		textRU, _ := value.GetString()

		value, _ = model.GetValue(iter, columnEN)
		textEN, _ := value.GetString()

		if (textRU == "") && (textEN != "") {
			win.filterChildEndIter = iter
			return true
		} else {
			return false
		}
	case filterNotOriginal:
		if win.userFilter.GetVisible() {
			win.userFilter.SetVisible(false)
		}

		// Фильтр записей без оригинала
		value, _ := model.GetValue(iter, columnRU)
		textRU, _ := value.GetString()

		value, _ = model.GetValue(iter, columnEN)
		textEN, _ := value.GetString()

		if (textRU != "") && (textEN == "") {
			win.filterChildEndIter = iter
			return true
		} else {
			return false
		}
	case filterUserFilter:
		//Пользовательский фильтр
		//Запускается по ComboFilter и по активации userFilter
		// Поэтому если ComboFilter - то просто включаем поле, если поле уже включено, фильтруем
		if win.userFilter.GetVisible() {

			filter, _ := win.userFilter.GetText()
			// Если фильтр пуст, выводим все
			if len(filter) == 0 {
				win.filterChildEndIter = iter
				return true
			}

			filter = str.ToUpper(filter)

			value, _ := model.GetValue(iter, columnRU)
			textRU, _ := value.GetString()

			value, _ = model.GetValue(iter, columnEN)
			textEN, _ := value.GetString()

			if str.Contains(str.ToUpper(textRU), filter) || str.Contains(str.ToUpper(textEN), filter) {
				win.filterChildEndIter = iter
				return true
			} else {
				return false
			}

		} else {
			win.userFilter.SetVisible(true)
		}
	}
	return true
}

func (win *MainWindow) ComboFilter_clicked() {
	win.combo_filter.SetSensitive(false)
	win.Filter.Refilter()
	win.combo_filter.SetSensitive(true)
}

func (win *MainWindow) ToolBtnTmpl_clicked() {

	wintmpl := tmpl.TmplWindowCreate()
	wintmpl.Col_SourceLang.SetTitle(win.Project.GetSourceLang())
	wintmpl.Col_TargetLang.SetTitle(win.Project.GetTargetLang())

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

		tmpl.SaveTmplToFile(TmplList, win.tmplFile)

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

// Сохраняем перевод
func (win *MainWindow) SaveTarget(outfile string) {
	var sum_all, sum_ru int //Подсчет % перевода
	sum_all = 0
	sum_ru = 0

	id := win.Project.GetHeaderNbyName("id")
	text := win.Project.GetHeaderNbyName("text")
	mode := win.Project.GetHeaderNbyName("mode")

	line := make([]string, 3)
	outdata := list.New()

	iter, _ := win.ListStore.GetIterFirst()
	next := true
	for next {
		valueId, err := win.ListStore.GetValue(iter, columnID)
		errorCheck(err)
		valueMode, err := win.ListStore.GetValue(iter, columnMode)
		errorCheck(err)
		valueRu, err := win.ListStore.GetValue(iter, columnRU)
		errorCheck(err)

		line[id], _ = valueId.GetString()
		line[mode], _ = valueMode.GetString()
		if line[mode] == "ucdt" {
			val, _ := valueRu.GetString()
			line[text] = str.ReplaceAll(val, "\t", " ")
		} else {
			line[text], _ = valueRu.GetString()
		}

		//Подсчет % перевода
		if line[mode] != "ucdn" {
			sum_all += 1
			if line[text] != "" {
				sum_ru += 1
			}
		}

		if win.clearNotOriginal {
			// Если есть перевод, а в оригинале такой строчки нет - пропускаем
			log.Println("[INFO]\tНе сохраняем строку перевода при отсутствии записи в оригинале")
			valueEn, err := win.ListStore.GetValue(iter, columnEN)
			errorCheck(err)
			strEN, _ := valueEn.GetString()
			if len(line[text]) != 0 && len(strEN) == 0 {
				next = win.ListStore.IterNext(iter)
				continue
			}

		}

		// Если русского перевода нет, и это текстовая строка (ucdt), пропускаем. Может быть(ucgt)
		//if line.Text == "" && line.Mode == "ucdt" {
		if line[text] == "" && line[mode] != "ucdn" {
			next = win.ListStore.IterNext(iter)
			continue
		}

		outdata.PushBack(line)
		line = make([]string, 3)

		// Проверка перевода на ошибки
		err = potbs.ValidateTranslate(line[text])
		if err != nil {
			log.Printf("[Warn]\tid[%s]: %s\n", line[id], err.Error())
		}

		// Проверка на Source=Target
		// Отключена т.к. много ложных. TODO
		// valueSource, err := win.ListStore.GetValue(iter, columnEN)
		// errorCheck(err)
		// sourceText, _ := valueSource.GetString()
		// if line.Mode == "ucdt" && line.Text == sourceText && line.Text != "String ID Not Found" && line.Text != ". . ." && line.Text[0:1] != "/" {
		// 	log.Printf("[warn]\tSource=Target ID: %s\n", line.Id)
		// }

		next = win.ListStore.IterNext(iter)

	}

	err := win.Project.SaveFile(outfile, outdata)
	errorCheck(err)
	log.Printf("[INFO]\tФайл %s сохранен.\n", outfile)

	log.Printf("[INFO]\tПереведено %d из %d (%d%s)", sum_ru, sum_all, int((sum_ru*100)/sum_all), "%")
	log.Printf("[INFO]\tОсталось: %d строк", int(sum_all-sum_ru))
}

func saveXLSXfile(win *MainWindow, outfile string) {
	//Экспортирует перевод в XLSX

	var line Tlang

	file := xlsx.NewFile()
	// Создаем новый лист
	sheet, err := file.AddSheet(win.Project.GetTargetLang())
	errorCheck(err)

	// Заполняем заголовки
	row := sheet.AddRow()

	cell := row.AddCell()
	cell.Value = "ID"
	cell = row.AddCell()
	cell.Value = "TYPE"
	cell = row.AddCell()
	cell.Value = "Original"
	cell = row.AddCell()
	cell.Value = "Translate"

	//iter, _ := win.ListStore.GetIterFirst()
	iter, _ := win.Filter.GetIterFirst()
	next := true
	for next {
		//valueId, err := win.ListStore.GetValue(iter, columnID)
		valueId, err := win.Filter.GetValue(iter, columnID)
		errorCheck(err)
		valueMode, err := win.Filter.GetValue(iter, columnMode)
		errorCheck(err)
		valueEN, err := win.Filter.GetValue(iter, columnEN)
		errorCheck(err)
		valueRu, err := win.Filter.GetValue(iter, columnRU)
		errorCheck(err)

		line.id, _ = valueId.GetString()
		line.mode, _ = valueMode.GetString()
		line.source, _ = valueEN.GetString()
		if line.mode == "ucdt" {
			val, _ := valueRu.GetString()
			line.target = str.ReplaceAll(val, "\t", " ")
		} else {
			line.target, _ = valueRu.GetString()
		}

		// Заполняем XLSX
		row = sheet.AddRow()
		//row.WriteStruct(&line, -1)
		cell = row.AddCell()
		cell.Value = line.id
		cell = row.AddCell()
		cell.Value = line.mode
		cell = row.AddCell()
		cell.Value = line.source
		cell = row.AddCell()
		cell.Value = line.target

		next = win.Filter.IterNext(iter)

	}

	// Сохраням измененный файл
	err = file.Save(outfile)
	errorCheck(err)

}

func loadXLSXfile(win *MainWindow, xlsxfile string, importALL bool) {
	//Импортирует перевод из XLSX

	file, err := xlsx.OpenFile(xlsxfile)
	errorCheck(err)

	// открываем первый лист
	sheet := file.Sheets[0]
	if sheet == nil {
		log.Println("[ERR]\tНе найден лист с переводом")
		return
	}

	//Data := make([]Tlang, 0)
	Data := make(map[string]Tlang)

	var line Tlang
	var row *xlsx.Row

	//Подгружаем значения
	for i := 1; i < sheet.MaxRow; i++ {
		row, err = sheet.Row(i)
		errorCheck(err)
		if row != nil {
			line.id = row.GetCell(columnID).Value
			line.mode = row.GetCell(columnMode).Value
			line.source = row.GetCell(columnEN).Value
			line.target = row.GetCell(columnRU).Value
			// Добавляем только строки с переводом
			if line.target != "" {
				Data[line.id+line.mode] = line
			}
		}
	}
	log.Printf("[INFO]\tЗагружено из файла %d строк.", len(Data))

	// Вносим изменения в перевод
	iter, _ := win.ListStore.GetIterFirst()
	next := true
	for next {

		valueId, err := win.ListStore.GetValue(iter, columnID)
		errorCheck(err)
		line.id, _ = valueId.GetString()

		valueMode, err := win.ListStore.GetValue(iter, columnMode)
		errorCheck(err)
		line.mode, _ = valueMode.GetString()

		//ucdn - пустая строка. нет смысла проверять далее
		if line.mode == "ucdn" {
			next = win.ListStore.IterNext(iter)
			continue
		}

		valueEN, err := win.ListStore.GetValue(iter, columnEN)
		errorCheck(err)
		line.source, _ = valueEN.GetString()

		// Если импортируем только новые, проверяем перевод
		if !importALL {
			valueRu, err := win.ListStore.GetValue(iter, columnRU)
			errorCheck(err)
			// Если перевода нет, добавляем
			if text, _ := valueRu.GetString(); text == "" {
				if val, ok := Data[line.id+line.mode]; ok {
					// Т.к. id+mode не уникален
					if line.source == val.source {
						win.ListStore.SetValue(iter, columnRU, val.target)
						//log.Println("[INFO]\tДобавлен перевод для записи: " + val.id)
					} else {
						log.Printf("[WARN]\tПропускаем запись %s, текст оригинала не совпадает.", val.id)
					}
				}
			}

		} else {
			if val, ok := Data[line.id+line.mode]; ok {
				// Т.к. id+mode не уникален
				if line.source == val.source {
					win.ListStore.SetValue(iter, columnRU, val.target)
				}
				//log.Println("[INFO]\tДобавлен перевод для записи: " + val.id)
			}
		}

		next = win.ListStore.IterNext(iter)
	}
}

// Заполнение окна с переводом при клике на строку
func (win *MainWindow) lineSelected(dialog *DialogWindow) {
	_, win.Iterator, _ = win.LineSelection.GetSelected()

	value, err := win.Filter.GetValue(win.Iterator, columnEN)
	errorCheck(err)
	strEN, err := value.GetString()
	errorCheck(err)
	dialog.BufferEn.SetText(strEN)

	value, err = win.Filter.GetValue(win.Iterator, columnRU)
	errorCheck(err)
	strRU, err := value.GetString()
	errorCheck(err)
	dialog.BufferRu.SetText(strRU)

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
			log.Println("[ERR]\tневерный итератор Next")
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
	//log.Printf("Поиск '%s': ничего не найдено.\n", searchtext)
	return nil
}

// Обратный поиск
func (win *MainWindow) searchPrev(text string) *gtk.TreePath {

	_, iter, ok := win.LineSelection.GetSelected()
	if !ok {
		//Iter = &win.EndIterator
		iter, _ = win.Filter.ConvertChildIterToIter(win.filterChildEndIter)
	}

	searchtext := str.ToUpper(text)
	loop := 1
	for loop < 3 {
		// Берем предыдущую строку, если ее нет, значит дошли до начала - переходим к последнему итератору
		if !win.Filter.IterPrevious(iter) {
			//*Iter = win.EndIterator
			iter, _ = win.Filter.ConvertChildIterToIter(win.filterChildEndIter)
			loop += 1
		}

		if !win.ListStore.IterIsValid(win.Filter.ConvertIterToChildIter(iter)) {
			log.Println("[ERR]\tневерный итератор Prev")
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
	//log.Printf("Поиск '%s': ничего не найдено.\n", searchtext)
	return nil
}

// Сохранение настроек
func (dialog *DialogWindow) saveCfg() {
	w, h := dialog.Window.GetSize()
	cfg.Section("Translate").Key("width").SetValue(strconv.Itoa(w))
	cfg.Section("Translate").Key("height").SetValue(strconv.Itoa(h))

	// Позиция всегда по центру родителя. Поэтому ее не сохраняем
	cfg.SaveTo(cfgFile)

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

	//Если нечего переводить, выходим
	if text == "" {
		return
	}

	// Заменяем текст оригинала по шаблонам. Для более точного перевода
	for _, line := range TmplList {
		text = str.ReplaceAll(text, line.En, line.Ru)
	}

	// отправляем в гугл
	res, err := tr.Translate(text, dialog.sourceLang, dialog.targetLang)
	if err == nil {
		dialog.BufferRu.SetText(res)
	}
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

func langName(lang string) string {
	name := str.ToUpper(lang)
	switch name {
	case "RU", "EN", "FR", "DE", "ES":
		return name
	default:
		return ""
	}

	return ""
}
