package ui

import (
	"log"
	str "strings"

	tr "github.com/bas24/googletranslatefree"
	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/gtk"
	"github.com/snakesel/potbs_langui/pkg/gtkutils"
	"github.com/snakesel/potbs_langui/pkg/locales"
	"github.com/snakesel/potbs_langui/pkg/tmpl"
)

const (
	dialogGlade = "data/dialog.glade"
)

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
	SourceLang string // исходный язык для перевода
	TargetLang string // на какой будем переводить

	TmplList *[]tmpl.TTmpl
}

// Окно диалога
func DialogWindowNew() *DialogWindow {

	// Создаём билдер
	b, err := gtk.BuilderNewFromFile(dialogGlade)
	errorCheck(err, "Error: No load dialog.glade")

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

// Заменяем текст оригинала по шаблонам
func (dialog *DialogWindow) BtnTmplRun_clicked() {

	text, err := dialog.BufferEn.GetText(dialog.BufferEn.GetStartIter(), dialog.BufferEn.GetEndIter(), true)
	errorCheck(err)

	for _, line := range *dialog.TmplList {
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
	for _, line := range *dialog.TmplList {
		text = str.ReplaceAll(text, line.En, line.Ru)
	}

	// отправляем в гугл
	res, err := tr.Translate(text, dialog.SourceLang, dialog.TargetLang)
	if err == nil {
		dialog.BufferRu.SetText(res)
	}
}

// Применение выбранного языка
func (dialog *DialogWindow) SetLocale(locale *locales.Printer) {
	dialog.Window.SetTitle(locale.Sprintf("DialogTitle"))
	dialog.BtnTmplRun.SetLabel(locale.Sprintf("from template"))
	dialog.BtnGooglTr.SetTooltipText(locale.Sprintf("Translate via Google Translate"))

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
