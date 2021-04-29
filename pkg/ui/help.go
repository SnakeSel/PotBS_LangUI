package ui

import (
	"bufio"
	//"fmt"
	"os"
	"regexp"
	str "strings"

	//"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/gtk"
	"github.com/gotk3/gotk3/pango"
)

type HelpWindow struct {
	Window *gtk.Window

	TextView     *gtk.TextView
	TextBuffer   *gtk.TextBuffer
	TextTagTable *gtk.TextTagTable

	BtnExit *gtk.Button
}

func HelpWindowNew() *HelpWindow {
	var err error

	win := new(HelpWindow)

	// Create a new toplevel window, set its title, and connect it to the
	// "destroy" signal to exit the GTK main loop when it is destroyed.
	win.Window, err = gtk.WindowNew(gtk.WINDOW_TOPLEVEL)
	checkErr(err, "Unable to create window")

	win.Window.SetTitle("Help")
	win.Window.Connect("destroy", func() {
		gtk.MainQuit()
	})

	// Получаем остальные объекты MainWindow
	win.TextTagTable, err = gtk.TextTagTableNew()
	checkErr(err)
	win.TextBuffer, err = gtk.TextBufferNew(win.TextTagTable)
	checkErr(err)
	addTags(win.TextBuffer)

	win.TextView, err = gtk.TextViewNewWithBuffer(win.TextBuffer)
	checkErr(err)
	win.TextView.SetEditable(false)         // запрет редактирования
	win.TextView.SetWrapMode(gtk.WRAP_WORD) // перенос строки по ширене

	win.BtnExit, err = gtk.ButtonNewWithLabel("Exit")
	checkErr(err)

	win.BtnExit.Connect("clicked", func() {
		win.Window.Close()
	})

	// построение UI
	mainbox, err := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 2)
	checkErr(err)

	scroll, err := gtk.ScrolledWindowNew(nil, nil)
	scroll.Add(win.TextView)
	scroll.SetVExpand(true) //расширяемость по вертикали
	mainbox.Add(scroll)

	boxButtons, err := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 1)
	checkErr(err)
	mainbox.Add(boxButtons)

	// Кнопки
	boxButtons.Add(win.BtnExit)

	boxButtons.SetHAlign(gtk.ALIGN_END) // расположение элементов по горизонтали
	boxButtons.SetSpacing(10)           // интервал между элементами
	boxButtons.SetHomogeneous(true)

	win.BtnExit.SetHAlign(gtk.ALIGN_END)

	//
	win.Window.Add(mainbox)

	// Set the default window size.
	win.Window.SetDefaultSize(800, 600)
	win.Window.SetPosition(gtk.WIN_POS_CENTER)

	return win
}

// Загружаем файл справки
func (win *HelpWindow) LoadHelpFile(filepach string) error {
	var text string

	// проверяем на существование
	_, err := os.Stat(filepach)
	if err != nil {
		return err
	}

	file, err := os.Open(filepach)
	if err != nil {
		return err
	}
	defer file.Close()

	win.TextBuffer.SetText("")
	line := 0

	scanner := bufio.NewScanner(file)

	for scanner.Scan() {

		iter := win.TextBuffer.GetEndIter()

		// Обрабатываем заголовки
		strings := str.SplitN(scanner.Text(), " ", 2)
		switch strings[0] {
		case "####":
			win.TextBuffer.InsertWithTagByName(iter, strings[1]+"\n", "tagH4")
			line += 1
			continue
		case "###":
			win.TextBuffer.InsertWithTagByName(iter, strings[1]+"\n", "tagH3")
			line += 1
			continue
			//default:
			//	win.TextBuffer.InsertAtCursor(scanner.Text() + "\n")

		}

		// Список
		switch {
		case str.HasPrefix(scanner.Text(), "* "):
			text = str.Replace(scanner.Text(), "* ", "• ", 1)
		case str.HasPrefix(scanner.Text(), "  * "):
			text = str.Replace(scanner.Text(), "  * ", "  ‐ ", 1)
		default:
			text = scanner.Text()
		}

		win.TextBuffer.Insert(iter, text)

		//	searchInLineCode
		re := regexp.MustCompile("`(.+?)`")
		varText := text
		// Ищем совпадение с регуляркой
		for codeBloc := re.FindStringIndex(varText); codeBloc != nil; codeBloc = re.FindStringIndex(varText) {
			//Получаем итераторы начала и конца блока
			startIter := win.TextBuffer.GetIterAtLineIndex(line, codeBloc[0])
			endIter := win.TextBuffer.GetIterAtLineIndex(line, codeBloc[1])
			// Вешаем тег
			win.TextBuffer.ApplyTagByName("tagInLineCode", startIter, endIter)
			// Удаляем символы разметки
			win.TextBuffer.Delete(startIter, win.TextBuffer.GetIterAtLineIndex(line, codeBloc[0]+1))
			win.TextBuffer.Delete(win.TextBuffer.GetIterAtLineIndex(line, codeBloc[1]-2), win.TextBuffer.GetIterAtLineIndex(line, codeBloc[1]-1))
			// Получаем актуальный текст
			varText, _ = win.TextBuffer.GetText(win.TextBuffer.GetIterAtLine(line), win.TextBuffer.GetEndIter(), false)
			//varText = str.Replace(varText, "`", "", 2)
		}

		//	searchBold
		re = regexp.MustCompile(`\*(.+?)\*`)
		varText, _ = win.TextBuffer.GetText(win.TextBuffer.GetIterAtLine(line), win.TextBuffer.GetEndIter(), false)
		// Ищем совпадение с регуляркой
		for codeBloc := re.FindStringIndex(varText); codeBloc != nil; codeBloc = re.FindStringIndex(varText) {
			//Получаем итераторы начала и конца блока
			startIter := win.TextBuffer.GetIterAtLineIndex(line, codeBloc[0])
			endIter := win.TextBuffer.GetIterAtLineIndex(line, codeBloc[1])
			// Вешаем тег
			win.TextBuffer.ApplyTagByName("tagBold", startIter, endIter)
			// Удаляем символы разметки
			win.TextBuffer.Delete(startIter, win.TextBuffer.GetIterAtLineIndex(line, codeBloc[0]+1))
			win.TextBuffer.Delete(win.TextBuffer.GetIterAtLineIndex(line, codeBloc[1]-2), win.TextBuffer.GetIterAtLineIndex(line, codeBloc[1]-1))
			// Получаем актуальный текст
			varText, _ = win.TextBuffer.GetText(win.TextBuffer.GetIterAtLine(line), win.TextBuffer.GetEndIter(), false)
		}

		win.TextBuffer.Insert(win.TextBuffer.GetEndIter(), "\n")
		line += 1

		// //	searchInLineCode
		// re := regexp.MustCompile("`(.+?)`")
		// codeBlocs := re.FindAllStringIndex(text, -1)

		// if len(codeBlocs) != 0 {

		// pos := 0
		// for _, block := range codeBlocs {
		// //fmt.Println(text[block[0]:block[1]])
		// win.TextBuffer.InsertAtCursor(text[pos:block[0]])
		// iter := win.TextBuffer.GetEndIter()
		// win.TextBuffer.InsertWithTagByName(iter, text[block[0]+1:block[1]-1], "tagInLineCode")
		// pos = block[1]
		// }
		// win.TextBuffer.InsertAtCursor(text[pos:] + "\n")
		// } else {
		// win.TextBuffer.InsertAtCursor(text + "\n")
		// }

	}

	return nil

}

func (win *HelpWindow) Run() {

	// Initialize GTK without parsing any command line arguments.
	gtk.Init(nil)

	// Recursively show all widgets contained in this window.
	win.Window.ShowAll()

	//Begin executing the GTK main loop.  This blocks until
	//gtk.MainQuit() is run.
	gtk.Main()

}

func addTags(tb *gtk.TextBuffer) {
	tb.CreateTag("tagH1", map[string]interface{}{
		"scale":              1.6,
		"weight":             pango.WEIGHT_BOLD,
		"pixels-above-lines": 4,
		"pixels-below-lines": 2,
	})
	tb.CreateTag("tagH2", map[string]interface{}{
		"scale":              1.5,
		"weight":             pango.WEIGHT_BOLD,
		"pixels-above-lines": 3,
		"pixels-below-lines": 2,
	})
	tb.CreateTag("tagH3", map[string]interface{}{
		"scale":              1.4,
		"weight":             pango.WEIGHT_BOLD,
		"pixels-above-lines": 2,
		"pixels-below-lines": 2,
	})
	tb.CreateTag("tagH4", map[string]interface{}{
		"scale":              1.3,
		"weight":             pango.WEIGHT_BOLD,
		"pixels-above-lines": 2,
		"pixels-below-lines": 1,
	})
	tb.CreateTag("tagH5", map[string]interface{}{
		"scale":  1.2,
		"weight": pango.WEIGHT_BOLD,
	})

	tb.CreateTag("tagInLineCode", map[string]interface{}{
		"family":     "Sans",
		"style":      pango.STYLE_ITALIC,
		"background": "WhiteSmoke", //WhiteSmoke "AliceBlue"
		//"background-rgba": *gdk.NewRGBA(214, 214, 214, 0.5),
		//"background-set": true,
	})
	tb.CreateTag("tagBold", map[string]interface{}{
		"weight": pango.WEIGHT_BOLD,
	})

}
