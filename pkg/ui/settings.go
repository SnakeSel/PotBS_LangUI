package ui

import (
	//"log"
	//str "strings"

	//"github.com/gotk3/gotk3/gdk"
	"fmt"

	"github.com/gotk3/gotk3/gtk"
	"github.com/snakesel/potbs_langui/pkg/gtkutils"
	"github.com/snakesel/potbs_langui/pkg/locales"
	"gopkg.in/ini.v1"
)

const (
	settingsGlade = "data/ui/settings.glade"
)

// startup id
const (
	startup_autoload = iota
	startup_opendialog
)

type SettingsWindow struct {
	Window *gtk.Window

	LblLang      *gtk.Label
	lblStarup    *gtk.Label
	ComboLang    *gtk.ComboBoxText
	ComboStartup *gtk.ComboBoxText
	CheckBtnLog  *gtk.CheckButton
	LblLogFile   *gtk.Label
	EntryLog     *gtk.Entry
	BtnOk        *gtk.Button
	BtnCancel    *gtk.Button

	LblProjLang     *gtk.Label
	LblSourceLang   *gtk.Label
	LblTargetLang   *gtk.Label
	ComboSourceLang *gtk.ComboBoxText
	ComboTargetLang *gtk.ComboBoxText

	AllLocaleName map[string]string
}

func SettingsWindowNew() *SettingsWindow {

	// Создаём билдер
	b, err := gtk.BuilderNewFromFile(settingsGlade)
	errorCheck(err, "Error: No load tmpl.glade")

	win := new(SettingsWindow)

	// Получаем объект главного окна по ID
	obj, err := b.GetObject("win_settings")
	errorCheck(err, "Error: No find win_settings")

	win.Window = obj.(*gtk.Window)

	// Получаем остальные объекты window_main
	win.LblLang = gtkutils.GetLabel(b, "lbl_lang")
	win.lblStarup = gtkutils.GetLabel(b, "lbl_startup")
	win.ComboLang = gtkutils.GetComboBoxText(b, "comboLang")
	win.ComboStartup = gtkutils.GetComboBoxText(b, "comboStartup")
	win.CheckBtnLog = gtkutils.GetCheckButton(b, "CheckBtnLog")
	win.LblLogFile = gtkutils.GetLabel(b, "LblLogFile")
	win.EntryLog = gtkutils.GetEntry(b, "EntryLog")
	win.BtnOk = gtkutils.GetButton(b, "BtnOk")
	win.BtnCancel = gtkutils.GetButton(b, "BtnCancel")

	win.LblProjLang = gtkutils.GetLabel(b, "lbl_projectLang")
	win.LblSourceLang = gtkutils.GetLabel(b, "lbl_sourceLang")
	win.LblTargetLang = gtkutils.GetLabel(b, "lbl_targetLang")
	win.ComboSourceLang = gtkutils.GetComboBoxText(b, "combo_sourceLang")
	win.ComboTargetLang = gtkutils.GetComboBoxText(b, "combo_targetLang")

	win.BtnCancel.Connect("clicked", func() {
		win.Window.Close()
	})

	win.CheckBtnLog.Connect("clicked", func() {
		logfile, _ := win.EntryLog.GetText()
		if logfile == "" {
			win.EntryLog.SetText("potbs_langui.log")
		}
	})
	// win.TreeView.Connect("row-activated", func() {
	// 	win.lineSelected()
	// })

	//win.Locales = make(map[string]string)
	return win
}

func (win *SettingsWindow) LoadCfg(cfg *ini.File) error {
	// Загрузка доступных языков
	currentLocale := cfg.Section("Main").Key("Language").MustString("")

	for k, v := range win.AllLocaleName {
		localelabel := fmt.Sprintf("%s (%s)", v, k)
		// Если выбранный язык совпадает с текущим добавляемым,
		// то делаем эту строку активной
		win.ComboLang.Append(k, localelabel)
		if k == currentLocale {
			win.ComboLang.SetActiveID(k)
		}

	}

	// Загрузка опции startup
	currentStartup := cfg.Section("Main").Key("Startup").MustInt(-1)
	if currentStartup != -1 {
		win.ComboStartup.SetActive(currentStartup)
	}

	// Загрузка лога
	logfile := cfg.Section("Main").Key("Log").MustString("")
	if logfile != "" {
		win.CheckBtnLog.SetActive(true)
		win.EntryLog.SetText(logfile)
	} else {
		win.CheckBtnLog.SetActive(false)
		//win.EntryLog.SetEditable(false)

	}

	// Загрузка проект-язык
	sourcelang := cfg.Section("Project").Key("SourceLang").MustString("AUTO")
	targetlang := cfg.Section("Project").Key("TargetLang").MustString("AUTO")

	win.ComboSourceLang.SetActiveID(sourcelang)
	win.ComboTargetLang.SetActiveID(targetlang)

	return nil
}

func (win *SettingsWindow) SetAllLocaleName(locales map[string]string) {
	win.AllLocaleName = locales
}

func (win *SettingsWindow) Run() {

	win.Window.Show()

}

// Применение выбранного языка
func (win *SettingsWindow) SetLocale(locale *locales.Printer) {
	win.ComboStartup.InsertText(startup_autoload, locale.Sprintf("open the latest project"))
	win.ComboStartup.InsertText(startup_opendialog, locale.Sprintf("file selection dialog"))
	win.LblLang.SetLabel(locale.Sprintf("Language") + ":")
	win.lblStarup.SetLabel(locale.Sprintf("Startup action") + ":")
	win.CheckBtnLog.SetLabel(locale.Sprintf("Save log file"))
	win.LblLogFile.SetLabel(locale.Sprintf("Log file") + ":")

	win.LblProjLang.SetLabel(locale.Sprintf("SettingProjLang"))
	win.LblSourceLang.SetLabel(locale.Sprintf("Source Lang") + ":")
	win.LblTargetLang.SetLabel(locale.Sprintf("Target Lang") + ":")

}
