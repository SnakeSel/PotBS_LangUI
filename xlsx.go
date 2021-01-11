package main

import (
	"log"
	str "strings"

	"path/filepath"

	"github.com/gotk3/gotk3/gtk"
	"github.com/snakesel/potbs_langui/pkg/gtkutils"
	"github.com/tealeg/xlsx"
)

func (win *MainWindow) ToolBtnExportXLSX_clicked() {
	native, err := gtk.FileChooserNativeDialogNew(win.locale.Sprintf("Select a file to save"), win.Window, gtk.FILE_CHOOSER_ACTION_SAVE, "OK", "Cancel")
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

	native, err := gtk.FileChooserNativeDialogNew(win.locale.Sprintf("Select the XLSX file to import"), win.Window, gtk.FILE_CHOOSER_ACTION_OPEN, "OK", "Cancel")
	errorCheck(err)

	native.SetCurrentFolder(filepath.Dir(win.targetFile))

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
	dlg.AddButton(win.locale.Sprintf("Not translated"), gtk.RESPONSE_ACCEPT)
	dlg.AddButton(win.locale.Sprintf("ALL"), gtk.RESPONSE_OK)
	dlg.AddButton(win.locale.Sprintf("Cancel"), gtk.RESPONSE_CANCEL)
	dlg.SetPosition(gtk.WIN_POS_CENTER)

	dlgBox, _ := dlg.GetContentArea()
	dlgBox.SetSpacing(6)

	lbl, _ := gtk.LabelNew(win.locale.Sprintf("Import from the first sheet in a book") + "!\n" + win.locale.Sprintf("Change only untranslated strings or all") + "?")
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
		line.id, _ = gtkutils.GetFilterValueString(win.Filter, iter, columnID)
		line.mode, _ = gtkutils.GetFilterValueString(win.Filter, iter, columnMode)
		line.source, _ = gtkutils.GetFilterValueString(win.Filter, iter, columnEN)
		if line.mode == "ucdt" {
			val, _ := gtkutils.GetFilterValueString(win.Filter, iter, columnRU)
			line.target = str.ReplaceAll(val, "\t", " ")
		} else {
			line.target, _ = gtkutils.GetFilterValueString(win.Filter, iter, columnRU)
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

		line.id, _ = gtkutils.GetListStoreValueString(win.ListStore, iter, columnID)
		line.mode, _ = gtkutils.GetListStoreValueString(win.ListStore, iter, columnMode)

		//ucdn - пустая строка. нет смысла проверять далее
		if line.mode == "ucdn" {
			next = win.ListStore.IterNext(iter)
			continue
		}

		line.source, _ = gtkutils.GetListStoreValueString(win.ListStore, iter, columnEN)

		// Если импортируем только новые, проверяем перевод
		if !importALL {
			// Если перевода нет, добавляем
			if text, _ := gtkutils.GetListStoreValueString(win.ListStore, iter, columnRU); text == "" {
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
