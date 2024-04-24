// potbs-langui

package main

import (
	"log"

	tr "github.com/snakesel/libretranslate"
	"github.com/snakesel/potbs_langui/pkg/apkstrings"
	"github.com/snakesel/potbs_langui/pkg/gtkutils"
	"github.com/snakesel/potbs_langui/pkg/locales"
	"github.com/snakesel/potbs_langui/pkg/potbs"
	"github.com/snakesel/potbs_langui/pkg/tmpl"
	"github.com/snakesel/potbs_langui/pkg/ui"

	"container/list"
	"path/filepath"

	"regexp"
	"sort"
	"strconv"
	str "strings"

	"github.com/gotk3/gotk3/glib"
	"github.com/gotk3/gotk3/gtk"

	"os"

	"fmt"

	// "runtime"
	// "runtime/pprof"

	"gopkg.in/ini.v1"
)

const (
	version      = "20230617"
	appId        = "snakesel.potbs-langui"
	MainGlade    = "data/ui/main.glade"
	tmplPatch    = "data/tmpl"
	cfgFile      = "data/cfg.ini"
	localesFile  = "data/locales"
	helpFilesDir = "data/help"
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

	GetModuleName() string

	ValidateTranslate(string, string) []error

	GetChecks() map[string]bool
	SetCheckActivebyName(name string, enable bool) error
	GetCheckDescriptionbyName(name string) (string, error)
}

// startup id
const (
	startup_autoload = iota
	startup_opendialog
)

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

	ToolBtnOpen       *gtk.ToolButton
	ToolBtnSave       *gtk.ToolButton
	ToolBtnSaveAs     *gtk.ToolButton
	ToolBtnSettings   *gtk.ToolButton
	ToolBtnTmpl       *gtk.ToolButton
	ToolBtnExportXLSX *gtk.ToolButton
	ToolBtnImportXLSX *gtk.ToolButton
	ToolBtnVerify     *gtk.ToolButton
	ToolBtnHelp       *gtk.ToolButton

	Renderer_ru *gtk.CellRendererText

	Iterator *gtk.TreeIter
	Project  intProject

	tmplFile           string        // Файл шаблонов для языка (tmplPatch_sourceLang-targetLang)
	filterChildEndIter *gtk.TreeIter // Хранит итератор последней записи. используется при обратном поиске
	clearNotOriginal   bool          // не сохранять строки которых нет в оригинале
	sourceFile         string
	targetFile         string

	locale *locales.Printer
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
		log.Fatal("[ERR]\tUnable to add row:", err.Error())
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
		dialog := ui.DialogWindowNew()

		// Map the handlers to callback functions, and connect the signals
		// to the Builder.
		signals := map[string]interface{}{
			"main_delete-event":            win.saveCfg, //Сохранение настроек при закрытии окна
			"main_btn_save_clicked":        win.ToolBtnSave_clicked,
			"main_btn_saveas_clicked":      win.ToolBtnSaveAs_clicked,
			"main_btn_export_xlsx_clicked": win.ToolBtnExportXLSX_clicked,
			"main_btn_import_xlsx_clicked": win.ToolBtnImportXLSX_clicked,
			"main_btn_tmpl_clicked":        win.ToolBtnTmpl_clicked,
			//"main_btn_Settings_clicked":    win.ToolBtnSettings_clicked,
			"main_btn_help_clicked":    win.ToolBtnHelp_clicked,
			"main_btn_verify_clicked":  win.ToolBtnVerify_clicked,
			"main_combo_filter_change": win.ComboFilter_clicked,
			"userfilter_activate":      win.ComboFilter_clicked,
			//"dialog_btn_tmpl_run_clicked":  dialog.BtnTmplRun_clicked,
			//"dialog_btn_googletr_clicked": dialog.BtnGoogleTr_clicked,
			//"dialog_btn_libretr_clicked":   dialog.BtnLibreTr_clicked,
		}
		b.ConnectSignals(signals)

		// Сигналы MainWindow
		win.Window.Connect("destroy", func() {
			w, h := dialog.Window.GetSize()
			cfg.Section("Translate").Key("width").SetValue(strconv.Itoa(w))
			cfg.Section("Translate").Key("height").SetValue(strconv.Itoa(h))

			// Позиция всегда по центру родителя. Поэтому ее не сохраняем
			cfg.SaveTo(cfgFile)
			application.Quit()
		})

		win.ToolBtnSettings.Connect("clicked", func() {
			win.ToolBtnSettings_clicked(dialog)
		})

		win.BtnDown.Connect("clicked", func() {
			win.BtnDown.SetSensitive(false)
			searchtext, _ := win.Search.GetText()
			patch := win.searchNext(searchtext)
			if patch != nil {
				win.TreeView.SetCursor(patch, nil, false)
			}
			win.BtnDown.SetSensitive(true)
		})

		win.BtnUp.Connect("clicked", func() {
			win.BtnUp.SetSensitive(false)
			searchtext, _ := win.Search.GetText()
			patch := win.searchPrev(searchtext)
			if patch != nil {
				win.TreeView.SetCursor(patch, nil, false)
			}
			win.BtnUp.SetSensitive(true)
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

		win.ToolBtnOpen.Connect("clicked", func() {
			// Получаем пути к файлам
			source, target, _, err := win.getFileNames(filepath.Dir(win.targetFile))
			if err != nil {
				return
			}

			//win.ListStore.Clear()

			// TODO FIX IT!!!!
			// т.к. ListStore.Clear() отрабатывает ОЧЕНЬ долго, пока просто создаем новый ListStore
			// но старый список ОСТАЕТСЯ В ПАМЯТИ

			// clear curren obj
			win.ListStore.Unref()
			win.Filter.Unref()

			win.filterChildEndIter = nil
			win.Iterator = nil

			b.Unref()

			// create new obj
			win.ListStore, err = gtk.ListStoreNew(glib.TYPE_STRING, glib.TYPE_STRING, glib.TYPE_STRING, glib.TYPE_STRING)
			errorCheck(err)

			win.Filter, _ = win.ListStore.TreeModel.FilterNew(nil)
			win.TreeView.SetModel(win.Filter)

			// Открываем перевод
			win.open(source, target)

			// Задаем фильтру правила фильтрации
			win.Filter.SetVisibleFunc(win.funcFilter)
			win.Filter.Refilter()

			// Указываем язык для окна диалаога (gtranslate)
			dialog.SourceLang = win.Project.GetSourceLang()
			dialog.TargetLang = win.Project.GetTargetLang()

			// Загружаем шаблоны
			// Имя файла шаблонов
			switch win.Project.GetModuleName() {
			case "potbs":
				win.tmplFile = tmplPatch + "_" + win.Project.GetSourceLang() + "-" + win.Project.GetTargetLang()
			case "apkstrings":
				win.tmplFile = tmplPatch + "_apkstrings_" + win.Project.GetSourceLang() + "-" + win.Project.GetTargetLang()

			}
			TmplList = tmpl.LoadTmplFromFile(win.tmplFile)

			dialog.TmplList = &TmplList

			// // Debug: pprof.WriteHeapProfile()
			// f, _ := os.Create("./memprofile2")
			// runtime.GC() // get up-to-date statistics
			// pprof.Lookup("heap").WriteTo(f, 0)
			// f.Close()

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

		dialog.BtnTmplRun.Connect("clicked", dialog.BtnTmplRun_clicked)
		dialog.BtnGooglTr.Connect("clicked", dialog.BtnGoogleTr_clicked)
		dialog.BtnLibreTr.Connect("clicked", dialog.BtnLibreTr_clicked)
		// Задать KEY и REGION для MS Translator
		//dialog.SetMSTranslatorKey(MSkey, MSregion)

		// ### применяем настроки
		win.Window.Resize(cfg.Section("Main").Key("width").MustInt(600), cfg.Section("Main").Key("height").MustInt(600))
		win.Window.Move(cfg.Section("Main").Key("posX").MustInt(0), cfg.Section("Main").Key("posY").MustInt(0))

		dialog.Window.Resize(cfg.Section("Translate").Key("width").MustInt(900), cfg.Section("Translate").Key("height").MustInt(300))

		win.clearNotOriginal = cfg.Section("Main").Key("ClearNotOriginal").MustBool(false)

		// #########################################
		//Язык программы

		win.locale, _ = locales.New(localesFile, cfg.Section("Main").Key("Language").MustString("en-US"))
		win.SetLocale()
		dialog.SetLocale(win.locale)

		// #########################################
		var source, target string
		// Получаем пути
		switch cfg.Section("Main").Key("Startup").MustInt(1) {
		case startup_autoload:
			source = cfg.Section("Project").Key("SourceFile").MustString("")
			target = cfg.Section("Project").Key("TargetFile").MustString("")
		case startup_opendialog:
			source = ""
			target = ""
		default:
			// Путь к файлам
			langFilePath := cfg.Section("Main").Key("Patch").MustString("")
			// Получаем пути к файлам
			source, target, _, err = win.getFileNames(langFilePath)
			if err != nil {
				win.Window.Close()
				log.Fatal(err.Error())
			}

		}

		// Открываем перевод
		win.open(source, target)

		// Задаем фильтру правила фильтрации
		win.Filter.SetVisibleFunc(win.funcFilter)
		win.Filter.Refilter()

		// Указываем язык для окна диалаога (gtranslate)
		dialog.SourceLang = win.Project.GetSourceLang()
		dialog.TargetLang = win.Project.GetTargetLang()

		// Загружаем шаблоны

		// Имя файла шаблонов
		switch win.Project.GetModuleName() {
		case "potbs":
			win.tmplFile = tmplPatch + "_" + win.Project.GetSourceLang() + "-" + win.Project.GetTargetLang()
		case "apkstrings":
			win.tmplFile = tmplPatch + "_apkstrings_" + win.Project.GetSourceLang() + "-" + win.Project.GetTargetLang()
		}
		TmplList = tmpl.LoadTmplFromFile(win.tmplFile)

		dialog.TmplList = &TmplList

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
	//win.ListStore = gtkutils.GetListStore(b, "liststore")
	win.ListStore, err = gtk.ListStoreNew(glib.TYPE_STRING, glib.TYPE_STRING, glib.TYPE_STRING, glib.TYPE_STRING)
	win.LineSelection = gtkutils.GetTreeSelection(b, "LineSelection")
	//win.Filter = gtkutils.GetTreeModelFilter(b, "treeFilter")
	win.Filter, _ = win.ListStore.TreeModel.FilterNew(nil)

	win.Renderer_ru = gtkutils.GetCellRendererText(b, "renderer_ru")

	win.Search = gtkutils.GetSearchEntry(b, "entry_search")
	win.Search_Full = gtkutils.GetCheckButton(b, "chk_full")
	win.combo_filter = gtkutils.GetComboBoxText(b, "combo_filter")
	win.userFilter = gtkutils.GetEntry(b, "entry_userfilter")

	win.ToolBtnOpen = gtkutils.GetToolButton(b, "tool_btn_open")
	win.ToolBtnSave = gtkutils.GetToolButton(b, "tool_btn_save")
	win.ToolBtnSaveAs = gtkutils.GetToolButton(b, "tool_btn_saveAs")
	win.ToolBtnSettings = gtkutils.GetToolButton(b, "tool_btn_settings")
	win.ToolBtnTmpl = gtkutils.GetToolButton(b, "tool_btn_tmpl")
	win.ToolBtnExportXLSX = gtkutils.GetToolButton(b, "tool_btn_export_xlsx")
	win.ToolBtnImportXLSX = gtkutils.GetToolButton(b, "tool_btn_import_xlsx")
	win.ToolBtnVerify = gtkutils.GetToolButton(b, "tool_btn_verify")
	win.ToolBtnHelp = gtkutils.GetToolButton(b, "tool_btn_help")

	win.BtnClose = gtkutils.GetButton(b, "button_close")
	win.BtnUp = gtkutils.GetButton(b, "btn_up")
	win.BtnDown = gtkutils.GetButton(b, "btn_down")

	win.TreeView.SetModel(win.Filter)

	return win
}

// Получить язык текста строк в столбце column
func getListStoreColumnLanguage(listStore *gtk.ListStore, column int) (string, error) {

	translate := tr.New(
		tr.Config{
			Url: "https://libretranslate.de",
		})

	var re = regexp.MustCompile(`[[:punct:]]`)
	var average float32 = 0.0
	lang := ""
	iter, ok := listStore.GetIterFirst()
	if !ok {
		return "", fmt.Errorf("Error GetIterFirs")
	}

	// Цикл пока уверенность в определении меньше 0.8
	for average < 80 && listStore.IterNext(iter) {
		text, err := gtkutils.GetListStoreValueString(listStore, iter, column)
		if err != nil {
			return "", fmt.Errorf("Error GetListStoreValueString: %s", err.Error())
		}
		// Убираем из строки [: :]
		reText := re.ReplaceAllString(text, "")
		// Если строка короткая, большпя вероятность ошибки, пропускаем ее
		if len(reText) < 30 {
			listStore.IterNext(iter)
			continue
		}
		//log.Println(reText)
		// Определяем язык текста
		score, newlang, err := translate.Detect(reText)
		if err != nil {
			//Если не смогли получить, выходим
			return "", fmt.Errorf("Error Detect Language")
		}

		//Если язык совпал, отлично
		if lang == newlang {
			average = average + (score / 3)
		} else {
			lang = newlang
			average = score / 3
		}
		//log.Printf("[DEBG]\tlang: %s %.2f avg:(%.2f)", newlang, score, average)

	}

	//log.Printf("[DEBG]\tResult lang: %s avg:(%.2f)", lang, average)
	return lang, nil
}

// Открываем перевод
func (win *MainWindow) open(sourceFile, targetFile string) {
	var fileExt string
	// проверяем файлы на существование
	_, err := os.Stat(sourceFile)
	_, err2 := os.Stat(targetFile)
	if err == nil && err2 == nil {
		fileExt = filepath.Ext(sourceFile)
		win.sourceFile = sourceFile
		win.targetFile = targetFile
	} else {
		// open source file
		win.sourceFile, win.targetFile, fileExt, err = win.getFileNames("")
		if err != nil {
			win.Window.Close()
			log.Fatal(err.Error())
		}
	}

	switch fileExt {
	case ".xml":
		log.Println("[INFO]\tUse apkstrings")
		win.Project = apkstrings.New(apkstrings.Config{})
	default:
		// Проект перевода
		win.Project = potbs.New(potbs.Config{
			//Debug:     os.Stdout,
			//Debug: log.Writer(),
		})

	}

	// Загружаем файлы перевода и выводим в таблицу
	err = win.loadListStore(win.sourceFile, win.targetFile)
	if err != nil {
		log.Fatalf("[ERR] %s", err.Error())
		os.Exit(1)
	}

	// Определяем Source Lang
	lang := cfg.Section("Project").Key("SourceLang").MustString("AUTO")

	switch str.ToUpper(lang) {
	case "AUTO":
		//log.Println("[DEBG]\tAUTO mode")

		lang, err = getListStoreColumnLanguage(win.ListStore, columnEN)

		// Если не нашли язык, к следующему выбору (берем из имени файла)
		if err == nil {
			break
		}
		fallthrough
	case "FILE":
		// Берем из имени файла (только для potbs)
		if win.Project.GetModuleName() != "potbs" {
			break
		}
		//log.Println("[DEBG]\tFILE mode")
		lang = filepath.Base(win.sourceFile)[0:2]
	}

	// Если получили валидное значение языка, ставим его
	switch str.ToUpper(lang) {
	case "RU", "EN", "DE", "ES", "FR":
		win.Project.SetSourceLang(str.ToUpper(lang))
	default:
		win.Project.SetSourceLang("Source")
	}

	// Определяем Target Lang

	lang = cfg.Section("Project").Key("TargetLang").MustString("AUTO")

	switch lang {
	case "AUTO":
		//log.Println("[DEBG]\tAUTO mode")

		lang, err = getListStoreColumnLanguage(win.ListStore, columnRU)

		// Если не нашли язык, к следующему выбору (берем из имени файла)
		if err == nil {
			break
		}
		fallthrough

	case "FILE":

		// Берем из имени файла (только для potbs)
		if win.Project.GetModuleName() != "potbs" {
			break
		}
		//log.Println("[DEBG]\tFILE mode")
		lang = filepath.Base(win.targetFile)[0:2]
	}

	// Если получили валидное значение языка, ставим его
	switch str.ToUpper(lang) {
	case "RU", "EN", "DE", "ES", "FR":
		win.Project.SetTargetLang(str.ToUpper(lang))
	default:
		win.Project.SetTargetLang("Target")
	}

	log.Printf("[INFO]\tSource file lang: %s\n", win.Project.GetSourceLang())
	log.Printf("[INFO]\tTarget file lang: %s\n", win.Project.GetTargetLang())

	// Устанавливаем заголовки полей
	win.TreeView.GetColumn(columnEN).SetTitle(win.Project.GetSourceLang())
	win.TreeView.GetColumn(columnRU).SetTitle(win.Project.GetTargetLang())

}

// Запускает диалог выбора source и target файлов
// filePath - директория выбора файлов
func (win *MainWindow) getFileNames(filePath string) (sourceName, targetName, extName string, err error) {
	//Функция создания и выполнения
	fileChooserDialog := func(title string) (string, error) {
		filter_dat, err := gtk.FileFilterNew()
		errorCheck(err)

		switch extName {
		case ".xml":
			filter_dat.AddPattern("strings.xml")
			filter_dat.SetName("strings.xml")
		case ".dat":
			filter_dat.AddPattern("*.dat")
			filter_dat.SetName(".dat")
		default:
			filter_dat.AddPattern("*.dat")
			filter_dat.AddPattern("strings.xml")
			filter_dat.SetName("All Supported")
		}

		filter_all, err := gtk.FileFilterNew()
		errorCheck(err)
		filter_all.AddPattern("*")
		filter_all.SetName("Any files")

		native, err := gtk.FileChooserNativeDialogNew(title, win.Window, gtk.FILE_CHOOSER_ACTION_OPEN, "OK", "Cancel")
		errorCheck(err)

		if filePath != "" {
			native.SetCurrentFolder(filePath)
		}
		native.AddFilter(filter_dat)
		native.AddFilter(filter_all)
		native.SetFilter(filter_dat)

		respons := native.Run()

		// NativeDialog возвращает int с кодом ответа. -3 это GTK_RESPONSE_ACCEPT
		if respons != int(gtk.RESPONSE_ACCEPT) {
			//win.Window.Close()
			//log.Fatal("[INFO]\tОтмена выбора файла")
			return "", fmt.Errorf("Отмена выбора файла")

		}

		filename := native.GetFilename()
		native.Destroy()

		return filename, nil
	}

	// open source file
	sourceName, err = fileChooserDialog(win.locale.Sprintf("SelectSourceFile"))
	if err != nil {
		return "", "", "", err
	}
	filePath = filepath.Dir(sourceName) //Нужен чтобы окно выбора target открылось там же
	extName = filepath.Ext(sourceName)

	//log.Println(extName)

	// open target file
	targetName, err = fileChooserDialog(win.locale.Sprintf("SelectTargetFile"))
	if err != nil {
		return "", "", "", err
	}

	return sourceName, targetName, extName, nil
}

func (win *MainWindow) loadListStore(sourceName, targetName string) error {

	id := win.Project.GetHeaderNbyName("id")
	mode := win.Project.GetHeaderNbyName("mode")
	text := win.Project.GetHeaderNbyName("text")

	var lang Tlang
	DataALL := make(map[string]Tlang)

	Data, err := win.Project.LoadFile(sourceName)
	if err != nil {
		return err
	}

	for e := Data.Front(); e != nil; e = e.Next() {
		line := e.Value.([]string)

		lang.id = line[id]

		if mode != -1 {
			lang.mode = line[mode]
		} else {
			lang.mode = ""
		}

		lang.source = line[text]
		//DataALL[line.Id+line.Mode] = lang

		// Проверяем, если уже есть такой id, добавляем _ (т.к. id+mode не уникален)
		if _, ok := DataALL[lang.id+lang.mode]; ok {
			DataALL[lang.id+lang.mode+"_"] = lang
		} else {
			DataALL[lang.id+lang.mode] = lang
		}
	}

	log.Printf("[INFO]\t%s успешно загружен", filepath.Base(sourceName))

	// Load target file

	Data, err = win.Project.LoadFile(targetName)
	if err != nil {
		return err
	}

	tmpmap := make(map[string]bool)
	for e := Data.Front(); e != nil; e = e.Next() {
		line := e.Value.([]string)

		lang.id = line[id]
		if mode != -1 {
			lang.mode = line[mode]
		} else {
			lang.mode = ""
		}
		lang.target = line[text]
		//DataALL[line.Id+line.Mode] = lang

		// Проверяем, если уже есть такой id, добавляем _ (т.к. id+mode не уникален)
		if _, ok := tmpmap[lang.id+lang.mode]; ok {
			lang.source = DataALL[lang.id+lang.mode+"_"].source
			DataALL[lang.id+lang.mode+"_"] = lang
			tmpmap[lang.id+lang.mode+"_"] = true
		} else {
			lang.source = DataALL[lang.id+lang.mode].source
			DataALL[lang.id+lang.mode] = lang
			tmpmap[lang.id+lang.mode] = true
		}
	}

	log.Printf("[INFO]\t%s успешно загружен", filepath.Base(targetName))

	//Сортируем
	lines := make([]Tlang, 0, len(DataALL))

	switch win.Project.GetModuleName() {
	case "apkstrings":
		keys := make([]string, 0, len(DataALL))
		for k := range DataALL {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		for _, k := range keys {

			lines = append(lines, DataALL[k])
		}
	default:
		for _, v := range DataALL {
			lines = append(lines, Tlang{v.id, v.mode, v.source, v.target})
		}
		sort.SliceStable(lines, func(i, j int) bool {
			before, _ := strconv.Atoi(lines[i].id)
			next, _ := strconv.Atoi(lines[j].id)
			return before < next
		})

	}

	//Выводим в таблицу
	for _, line := range lines {
		err = addRow(win.ListStore, line.id, line.mode, line.source, line.target)
		errorCheck(err)
	}

	// отключаем неиспользуемые столбцы
	if mode == -1 {
		win.TreeView.GetColumn(columnMode).SetVisible(false)
	}

	// Data = nil
	// DataALL = nil
	// lines = nil

	return nil
}

// Применение выбранного языка
func (win *MainWindow) SetLocale() {
	win.Window.SetTitle(win.locale.Sprintf("Title"))
	win.ToolBtnOpen.SetLabel(win.locale.Sprintf("New"))
	win.ToolBtnSave.SetLabel(win.locale.Sprintf("Save"))
	win.ToolBtnSaveAs.SetLabel(win.locale.Sprintf("SaveAs"))
	win.ToolBtnSettings.SetTooltipText(win.locale.Sprintf("Settings"))
	win.ToolBtnExportXLSX.SetLabel(win.locale.Sprintf("ExportXLSX"))
	win.ToolBtnImportXLSX.SetLabel(win.locale.Sprintf("ImportXLSX"))
	win.ToolBtnTmpl.SetLabel(win.locale.Sprintf("Template"))
	win.ToolBtnVerify.SetTooltipText(win.locale.Sprintf("Checking the translation to errors"))
	win.BtnClose.SetLabel(win.locale.Sprintf("Exit"))
	win.BtnUp.SetLabel(win.locale.Sprintf("Up"))
	win.BtnDown.SetLabel(win.locale.Sprintf("Down"))
	activeFilter := win.combo_filter.GetActive()
	win.combo_filter.RemoveAll()
	win.combo_filter.InsertText(filterALL, win.locale.Sprintf("ALL"))
	win.combo_filter.InsertText(filterNotTranslate, win.locale.Sprintf("Not Translated"))
	win.combo_filter.InsertText(filterNotOriginal, win.locale.Sprintf("Not Original"))
	win.combo_filter.InsertText(filterUserFilter, win.locale.Sprintf("User Filter"))
	win.combo_filter.SetActive(activeFilter)
	win.userFilter.SetPlaceholderText(win.locale.Sprintf("UserFilterPlaceholder"))
	win.Search.SetPlaceholderText(win.locale.Sprintf("SearchPlaceholder"))
	win.Search_Full.SetLabel(win.locale.Sprintf("SearchFull"))

}

// Сохранение настроек
func (win *MainWindow) saveCfg() {

	w, h := win.Window.GetSize()
	cfg.Section("Main").Key("width").SetValue(strconv.Itoa(w))
	cfg.Section("Main").Key("height").SetValue(strconv.Itoa(h))

	x, y := win.Window.GetPosition()
	cfg.Section("Main").Key("posX").SetValue(strconv.Itoa(x))
	cfg.Section("Main").Key("posY").SetValue(strconv.Itoa(y))

	cfg.Section("Main").Key("Patch").SetValue(filepath.Dir(win.targetFile))

	cfg.Section("Project").Key("SourceFile").SetValue(win.sourceFile)
	cfg.Section("Project").Key("TargetFile").SetValue(win.targetFile)

	cfg.SaveTo(cfgFile)
}

func (win *MainWindow) ToolBtnSave_clicked() {
	dialog := gtk.MessageDialogNew(win.Window, gtk.DIALOG_MODAL, gtk.MESSAGE_INFO, gtk.BUTTONS_OK_CANCEL, win.locale.Sprintf("Warning")+"!")
	dialog.FormatSecondaryText(win.locale.Sprintf("Are you sure you want to overwrite") + ":\n" + filepath.Base(win.targetFile) + " ?")
	resp := dialog.Run()
	dialog.Close()
	if resp == gtk.RESPONSE_OK {
		//win.SaveTarget(filepath.Join(win.langFilePath, win.langFileName))
		win.SaveTarget(win.targetFile)
		//win.Window.Destroy()
	}

}

func (win *MainWindow) ToolBtnSaveAs_clicked() {
	native, err := gtk.FileChooserNativeDialogNew(win.locale.Sprintf("Select a file to save"), win.Window, gtk.FILE_CHOOSER_ACTION_SAVE, "OK", "Cancel")
	errorCheck(err)
	native.SetCurrentFolder(filepath.Dir(win.targetFile))
	native.SetCurrentName(win.Project.GetTargetLang() + "_data_mod" + filepath.Ext(win.targetFile))
	resp := native.Run()

	if resp == int(gtk.RESPONSE_ACCEPT) {
		win.SaveTarget(native.GetFilename())
		//log.Printf("[INFO]\tФайл %s сохранен.\n", native.GetFilename())
		//win.Window.Destroy()
	}

}

// Фильтр
func (win *MainWindow) funcFilter(model *gtk.TreeModel, iter *gtk.TreeIter) bool {

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

		textRU, _ := gtkutils.GetTreeModelValueString(model, iter, columnRU)
		textEN, _ := gtkutils.GetTreeModelValueString(model, iter, columnEN)

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
		textRU, _ := gtkutils.GetTreeModelValueString(model, iter, columnRU)
		textEN, _ := gtkutils.GetTreeModelValueString(model, iter, columnEN)

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

			textRU, _ := gtkutils.GetTreeModelValueString(model, iter, columnRU)
			textEN, _ := gtkutils.GetTreeModelValueString(model, iter, columnEN)

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
	// Применяем локализацию
	wintmpl.SetLocale(win.locale)

	// Загружаем настройки
	wintmpl.Col_SourceLang.SetTitle(win.Project.GetSourceLang())
	wintmpl.Col_TargetLang.SetTitle(win.Project.GetTargetLang())

	wintmpl.Window.Resize(cfg.Section("Template").Key("width").MustInt(900), cfg.Section("Template").Key("height").MustInt(400))
	wintmpl.Window.Move(cfg.Section("Template").Key("posX").MustInt(0), cfg.Section("Template").Key("posY").MustInt(0))

	wintmpl.BtnSave.Connect("clicked", func() {
		TmplList = wintmpl.GetTmpls()

		// Сортируем от большего совпадения к меньшему
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

// Открывает окно настроек
func (win *MainWindow) ToolBtnSettings_clicked(dialog *ui.DialogWindow) {

	winSetings := ui.SettingsWindowNew()
	// Применяем локализацию
	winSetings.SetLocale(win.locale)

	winSetings.SetAllLocaleName(locales.GetAllLocaleName(localesFile))

	winSetings.LoadCfg(cfg)

	oldlocale := cfg.Section("Main").Key("Language").MustString("en-US")

	// Сигналы
	// Получаем значения при ОК
	winSetings.BtnOk.Connect("clicked", func() {
		cfg.Section("Main").Key("Startup").SetValue(strconv.Itoa(winSetings.ComboStartup.GetActive()))

		if winSetings.CheckBtnLog.Activate() {
			text, _ := winSetings.EntryLog.GetText()
			cfg.Section("Main").Key("Log").SetValue(text)
		} else {
			cfg.Section("Main").Key("Log").SetValue("")
		}

		cfg.Section("Main").Key("Language").SetValue(winSetings.ComboLang.GetActiveID())

		cfg.Section("Project").Key("SourceLang").SetValue(winSetings.ComboSourceLang.GetActiveID())
		cfg.Section("Project").Key("TargetLang").SetValue(winSetings.ComboTargetLang.GetActiveID())

		cfg.SaveTo(cfgFile)

		dialog.SourceLang = win.Project.GetSourceLang()
		dialog.TargetLang = win.Project.GetTargetLang()

		winSetings.Window.Close()
	})

	// Изменение языка при закрытии
	winSetings.Window.Connect("delete-event", func() {
		if oldlocale != winSetings.ComboLang.GetActiveID() {
			win.locale, _ = locales.New(localesFile, winSetings.ComboLang.GetActiveID())
			win.SetLocale()
			dialog.SetLocale(win.locale)
		}

	})

	winSetings.Run()
}

// Открывает окно справки
func (win *MainWindow) ToolBtnHelp_clicked() {
	winHelp := ui.HelpWindowNew()

	helpFileName := fmt.Sprintf("%s_%s", win.Project.GetModuleName(), cfg.Section("Main").Key("Language").MustString("en-US"))
	err := winHelp.LoadHelpFile(filepath.Join(helpFilesDir, helpFileName))
	if err != nil {
		if os.IsNotExist(err) {
			log.Printf("[DBG]\tHelp file \"%s\" not found", filepath.Join(helpFilesDir, helpFileName))
		}
		return
	}
	winHelp.Run()
}

// Открывает окно проверки перевода
func (win *MainWindow) ToolBtnVerify_clicked() {
	// Только для POTBS
	if win.Project.GetModuleName() != "potbs" {
		return
	}

	// Создаем окно
	winVerify := ui.VerifyWindowNew()
	// Переводим
	winVerify.SetLocale(win.locale)

	// Задаем имя файла игнорируемых
	winVerify.SetFileIgnoreErr("data/ignoreErr_" + win.Project.GetSourceLang() + "_" + win.Project.GetTargetLang())

	// Получаем список проверок
	allChecks := win.Project.GetChecks()

	// Сортируем и добавляем кнопки
	keys := make([]string, 0, len(allChecks))
	for k := range allChecks {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// Добавляем кнопки
	// for name, active := range allChecks {
	// 	label, _ := win.Project.GetCheckDescriptionbyName(name)
	// 	winVerify.AddCheckButton(name, win.locale.Sprintf(label), active)
	// }
	for _, name := range keys {
		label, _ := win.Project.GetCheckDescriptionbyName(name)
		winVerify.AddCheckButton(name, win.locale.Sprintf(label), allChecks[name])
	}

	// Запуск проверки
	winVerify.BtnVerify.Connect("clicked", func() {
		// очищаем список ошибок
		winVerify.ListStore.Clear()

		// Применяем значения проверок
		for name, _ := range allChecks {
			active, err := winVerify.GetCheckButtonActive(name)
			if err == nil {
				win.Project.SetCheckActivebyName(name, active)
			}
		}

		// Цикл по всем записям
		iter, _ := win.ListStore.GetIterFirst()
		next := true
		for next {
			//Получаем данные полей из ListStore
			id, err := gtkutils.GetListStoreValueString(win.ListStore, iter, columnID)
			errorCheck(err)

			mode, err := gtkutils.GetListStoreValueString(win.ListStore, iter, columnMode)
			errorCheck(err)

			if mode == "ucdn" {
				next = win.ListStore.IterNext(iter)
				continue
			}

			targetText, _ := gtkutils.GetListStoreValueString(win.ListStore, iter, columnRU)
			sourceText, _ := gtkutils.GetListStoreValueString(win.ListStore, iter, columnEN)

			// Если перевода нет - пропускаем.
			if targetText == "" || sourceText == "" {
				next = win.ListStore.IterNext(iter)
				continue
			}

			// Проверка перевода на ошибки
			listErr := win.Project.ValidateTranslate(sourceText, targetText)
			if listErr != nil {
				for _, err := range listErr {
					winVerify.AddRow(id, err.Error())
				}
			}

			// к следующей записи
			next = win.ListStore.IterNext(iter)

		}
	})

	// Переход к записи с ошибкой при клике
	winVerify.TreeView.Connect("row-activated", func() {

		id := winVerify.GetSelectedID()
		if id == "" {
			return
		}

		patch := win.searchNext(id)
		if patch != nil {
			win.TreeView.SetCursor(patch, nil, false)
		}

	})

	winVerify.Run()

}

// Сохраняем перевод
func (win *MainWindow) SaveTarget(outfile string) {

	var err error
	var sum_all, sum_ru int //Подсчет % перевода
	sum_all = 0
	sum_ru = 0

	// Позиция поля в списке
	id := win.Project.GetHeaderNbyName("id")
	text := win.Project.GetHeaderNbyName("text")
	mode := win.Project.GetHeaderNbyName("mode")

	// список из line
	outdata := list.New()

	// Цикл по всем записям
	iter, _ := win.ListStore.GetIterFirst()
	next := true
	for next {

		// массив строк, длина = кол-ву заголовков в плагине
		line := make([]string, win.Project.GetHeaderLen())

		//Получаем данные полей из ListStore
		line[id], err = gtkutils.GetListStoreValueString(win.ListStore, iter, columnID)
		errorCheck(err)
		//fmt.Println(line[id])
		// Если поле mode существует, заполняем
		// иначе заполняем перевод
		if mode != -1 {
			line[mode], err = gtkutils.GetListStoreValueString(win.ListStore, iter, columnMode)
			errorCheck(err)
		} else {
			// заполняем XML

			line[text], err = gtkutils.GetListStoreValueString(win.ListStore, iter, columnRU)
			errorCheck(err)
			sum_all += 1

			if line[text] != "" {
				sum_ru += 1
			} else {
				//перевода нет, идем к следующей строке
				next = win.ListStore.IterNext(iter)
				continue
			}

		}

		// Заполняем Potsb
		if win.Project.GetModuleName() == "potbs" {

			if line[mode] == "ucdt" {
				val, _ := gtkutils.GetListStoreValueString(win.ListStore, iter, columnRU)
				line[text] = str.ReplaceAll(val, "\t", " ")
			} else {
				line[text], _ = gtkutils.GetListStoreValueString(win.ListStore, iter, columnRU)
			}

			//Подсчет % перевода
			if line[mode] != "ucdn" {
				sum_all += 1
				if line[text] != "" {
					sum_ru += 1
				}
			}

			// Если русского перевода нет, и это текстовая строка (ucdt), пропускаем. Может быть(ucgt)
			//if line.Text == "" && line.Mode == "ucdt" {
			if line[text] == "" && line[mode] != "ucdn" {
				next = win.ListStore.IterNext(iter)
				continue
			}

		}

		// Если есть перевод, а в оригинале такой строчки нет - пропускаем. (при влюченной опции clearNotOriginal)
		if win.clearNotOriginal {
			log.Println("[INFO]\tНе сохраняем строку перевода при отсутствии записи в оригинале")
			strEN, _ := gtkutils.GetListStoreValueString(win.ListStore, iter, columnEN)
			if len(line[text]) != 0 && len(strEN) == 0 {
				next = win.ListStore.IterNext(iter)
				continue
			}
		}

		// // Проверка перевода на ошибки

		// sourceText, _ := gtkutils.GetListStoreValueString(win.ListStore, iter, columnEN)
		// //log.Printf("[%s]\n", line[id])
		// listErr := win.Project.ValidateTranslate(sourceText, line[text])
		// if listErr != nil {
		// 	for _, err := range listErr {
		// 		log.Printf("[Warn]\tid[%s]: %s\n", line[id], err.Error())
		// 	}
		// }

		// Добавляем Line в список
		outdata.PushBack(line)

		// к следующей записи
		next = win.ListStore.IterNext(iter)

	}

	err = win.Project.SaveFile(outfile, outdata)
	errorCheck(err)
	log.Printf("[INFO]\tФайл %s сохранен.\n", outfile)

	log.Printf("[INFO]\tПереведено %d из %d (%d%s)", sum_ru, sum_all, int((sum_ru*100)/sum_all), "%")
	log.Printf("[INFO]\tОсталось: %d строк", int(sum_all-sum_ru))
}

// Заполнение окна с переводом при клике на строку
func (win *MainWindow) lineSelected(dialog *ui.DialogWindow) {
	_, iter, ok := win.LineSelection.GetSelected()
	if !ok {
		log.Println("[err]\tGetSelected error iter")
	}

	strEN, err := gtkutils.GetFilterValueString(win.Filter, iter, columnEN)
	errorCheck(err)
	dialog.BufferEn.SetText(strEN)

	strRU, err := gtkutils.GetFilterValueString(win.Filter, iter, columnRU)
	errorCheck(err)
	dialog.BufferRu.SetText(strRU)

	strID, err := gtkutils.GetFilterValueString(win.Filter, iter, columnID)
	errorCheck(err)
	dialog.Label.SetText(strID)

	win.Iterator = iter

	//dialog.Window.Run()
	dialog.Window.Show()

}

// Прямой поиск
func (win *MainWindow) searchNext(text string) *gtk.TreePath {

	iter := new(gtk.TreeIter)

	_, childIter, ok := win.LineSelection.GetSelected()
	if !ok {
		//iter, _ = win.ListStore.GetIterFirst()
		iter, _ = win.Filter.GetIterFirst()

	} else {
		if win.Filter.IterHasChild(childIter) {
			iter, _ = win.Filter.ConvertChildIterToIter(childIter)
		} else {
			iter = childIter
		}
	}

	searchtext := str.ToUpper(text)
	// startIter, err := iter.Copy()
	// errorCheck(err)
	startIter := new(gtk.TreeIter)
	startIter.GtkTreeIter = iter.GtkTreeIter

	// Берем следующую строку, если ее нет, значит дошли до конца - переходим к первой
	//if !win.ListStore.IterNext(iter) {
	if !win.Filter.IterNext(iter) {
		iter, _ = win.Filter.GetIterFirst()
	}

	for startIter.GtkTreeIter != iter.GtkTreeIter {

		// if !win.ListStore.IterIsValid(win.Filter.ConvertIterToChildIter(iter)) {
		// 	log.Println("[ERR]\tневерный итератор Next")
		// 	continue
		// }

		//Ищем совпадения в текущей записи
		if gtkutils.FilterSearchTextfromIter(win.Filter, iter, searchtext, win.Search_Full.GetActive()) {
			patch, err := win.Filter.GetPath(iter)
			errorCheck(err)
			return patch
		}
		// Берем следующую строку, если ее нет, значит дошли до конца - переходим к первой
		//if !win.ListStore.IterNext(iter) {
		if !win.Filter.IterNext(iter) {
			log.Println("дошли до конца")
			iter, _ = win.Filter.GetIterFirst()
		}

	}

	log.Printf("Поиск '%s': ничего не найдено.\n", searchtext)

	msg := gtk.MessageDialogNew(win.Window, gtk.DIALOG_MODAL, gtk.MESSAGE_INFO, gtk.BUTTONS_CLOSE, fmt.Sprintf("%s\n\n%s", text, win.locale.Sprintf("Not found")))
	msg.Run()
	msg.Close()

	return nil
}

// Обратный поиск
func (win *MainWindow) searchPrev(text string) *gtk.TreePath {

	iter := new(gtk.TreeIter)

	_, childIter, ok := win.LineSelection.GetSelected()
	if !ok {
		iter, _ = win.Filter.ConvertChildIterToIter(win.filterChildEndIter)

	} else {
		if win.Filter.IterHasChild(childIter) {
			iter, _ = win.Filter.ConvertChildIterToIter(childIter)
		} else {
			iter = childIter
		}
	}

	searchtext := str.ToUpper(text)

	startIter := new(gtk.TreeIter)
	startIter.GtkTreeIter = iter.GtkTreeIter

	// Берем предыдущую строку, если ее нет, значит дошли до начала - переходим к последнему итератору
	if !win.Filter.IterPrevious(iter) {
		//*Iter = win.EndIterator
		iter, _ = win.Filter.ConvertChildIterToIter(win.filterChildEndIter)
	}

	for startIter.GtkTreeIter != iter.GtkTreeIter {
		// Берем предыдущую строку, если ее нет, значит дошли до начала - переходим к последнему итератору
		if !win.Filter.IterPrevious(iter) {
			//*Iter = win.EndIterator
			iter, _ = win.Filter.ConvertChildIterToIter(win.filterChildEndIter)
		}

		// if !win.ListStore.IterIsValid(win.Filter.ConvertIterToChildIter(iter)) {
		// 	log.Println("[ERR]\tневерный итератор Prev")
		// 	continue
		// }

		//Ищем совпадения в текущей записи
		if gtkutils.FilterSearchTextfromIter(win.Filter, iter, searchtext, win.Search_Full.GetActive()) {
			patch, err := win.Filter.GetPath(iter)
			errorCheck(err)
			return patch
		}

	}
	log.Printf("Поиск '%s': ничего не найдено.\n", searchtext)

	msg := gtk.MessageDialogNew(win.Window, gtk.DIALOG_MODAL, gtk.MESSAGE_INFO, gtk.BUTTONS_CLOSE, fmt.Sprintf("%s\n\n%s", text, win.locale.Sprintf("Not found")))
	msg.Run()
	msg.Close()

	return nil
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
