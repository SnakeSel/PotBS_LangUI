package main

import (
	"log"
	str "strings"

	"github.com/snakesel/potbs_langui/pkg/gtkutils"
	"github.com/tealeg/xlsx"
)

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
