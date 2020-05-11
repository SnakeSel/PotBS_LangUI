package potbs

import (
	"bufio"
	"bytes"
	"fmt"
	"os"

	s "strings"

	"regexp"
	"strconv"
)

var BeginByte = []byte("ï»¿")

type TData struct {
	Id   string
	Mode string
	Text string
}

type TDir struct {
	Id     int
	pos    int
	lenght int
}

// dropCR drops a terminal \r from the data.
func dropCR(data []byte) []byte {
	if len(data) > 0 && data[len(data)-1] == '\r' {
		return data[0 : len(data)-1]
	}

	return data
}

// Конец строки в виндовой кодировке (\r\n)
func scanCRLF(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}
	if i := bytes.Index(data, []byte{'\r', '\n'}); i >= 0 {
		// We have a full newline-terminated line.
		return i + 2, dropCR(data[0:i]), nil
	}
	// If we're at EOF, we have a final, non-terminated line. Return it.
	if atEOF {
		return len(data), dropCR(data), nil
	}
	// Request more data.
	return 0, nil, nil
}

// Поиск n-го вхождения substr в str
// Возвращает номер позиции или -1, если такого вхождения нет.
func indexN(str, substr string, n int) int {

	ind := 0
	pos := 0
	for i := 0; i < n; i++ {
		ind = s.Index(str[pos:], substr)
		if ind == -1 {
			return ind
		}
		pos += ind
		pos += len(substr)
	}

	return pos
}

func ReadDat(filepach string) []TData {

	// Входной dat файл со списком строк вида:
	// <id>\t<вид строки>\t<строка>\r\n
	// вид строки:
	// ucdt - текст. подсчитваем всю строку
	// ucdn - пустая строка
	// mcdt - Текст со скриптом. Далее строка имеет вид: <текст>\t<scriptID>\t<script name>. Подсчитваем только <текст>
	// mcdn - Пустая строка со скриптом

	Data := make([]TData, 0)

	file, err := os.Open(filepach)
	сheckErr(err)
	defer file.Close()

	scanner := bufio.NewScanner(file)

	var line TData
	var lineLen int
	var splitline []string

	// Концом строки считается \r\n
	scanner.Split(scanCRLF)
	lineN := 0
	for scanner.Scan() {
		// Подсчитваем длину и разбиваем строку по "\t"
		if lineN == 0 {
			// Первая строка сожержит заголовок.
			// Пока просто отбрасываем 6 байт
			lineall := scanner.Text()[6:]
			splitline = s.Split(lineall, "\t")
			lineLen = len(lineall)
		} else {
			splitline = s.SplitN(scanner.Text(), "\t", 3)
			lineLen = len(scanner.Bytes())
			//lineLen = utf8.RuneCount(scanner.Bytes())
		}

		// костыль пустой строки
		if lineLen == 0 {
			continue
		}

		line.Id = splitline[0]

		// Проверяем кол-во разделенных элементов. Должно быть 3
		if len(splitline) >= 3 {
			line.Mode = splitline[1]
			line.Text = splitline[2]
		} else if len(splitline) == 2 {
			// 2 быват при пустой строке (ucdn)
			line.Mode = splitline[1]
			line.Text = ""
		} else {
			// При строке с \r\n в середине
			line.Mode = "none"
			line.Text = ""
			continue
		}

		Data = append(Data, line)

		lineN += 1

	}

	return Data
}

func SaveDat(filepach string, Datas []TData) []TDir {
	dirs := make([]TDir, 0)
	var dir TDir
	var linelen int
	pos := 0 // Бит в файле

	filedat, err := os.Create(filepach)
	сheckErr(err, "Unable to create file: "+filepach)
	defer filedat.Close()

	for id, line := range Datas {
		if id == 0 {
			filedat.WriteString(fmt.Sprintf("%s%s\t%s\t%s\r\n", BeginByte, line.Id, line.Mode, line.Text))
			pos += 6 //BeginByte
		} else {
			filedat.WriteString(fmt.Sprintf("%s\t%s\t%s\r\n", line.Id, line.Mode, line.Text))
		}

		dir.Id, _ = strconv.Atoi(line.Id)
		dir.pos = pos

		// Расчитываем длину строки
		linelen = len(line.Id)
		linelen += 1 //\t
		linelen += len(line.Mode)
		linelen += 1 //\t
		mcdtlen := linelen
		linelen += len(line.Text)
		// Ебала с размером. В позицию идет вся длинна (linelen), а в размер только длина текста (mcdtlen).
		switch line.Mode {
		case "mcdt":
			// mcdt - Текст со скриптом. Далее строка имеет вид: <текст>\t<scriptID>\t<script name>. Подсчитваем только <текст>
			ind := s.Index(line.Text, "\t")
			// -1 - не найдено
			if ind == -1 {
				mcdtlen = linelen
			} else {
				mcdtlen += len(line.Text[:ind])
			}
			dir.lenght = mcdtlen
		case "mcdn":
			// mcdn - Пустая строка со скриптом. line.text не подсчитываем
			dir.lenght = mcdtlen
		default:
			dir.lenght = linelen
		}

		dirs = append(dirs, dir)

		// Увеличиваем позицию на длину строки + 2 бита на перенос
		pos += linelen
		pos += 2 //\r\n
	}

	return dirs

}

func SaveDir(filepach string, dirs []TDir) {
	filedir, err := os.Create(filepach)
	сheckErr(err, "Unable to create file: "+filepach)
	defer filedir.Close()

	// Записываем начало
	filedir.WriteString(fmt.Sprintf("## Count:\t%d\r\n", len(dirs)))
	//filedir.WriteString("## Locale:\tru_RU\r\n")

	for _, dir := range dirs {
		filedir.WriteString(fmt.Sprintf("%d\t%d\t%d\td\r\n", dir.Id, dir.pos, dir.lenght))
	}

}

func ValidateTranslate(translate string) error {

	//Проверяем перевод макросов
	re_macros := regexp.MustCompile(`\[\!(.+?)\!\]`)
	macros := re_macros.FindAllString(translate, -1)
	if len(macros) != 0 {
		for _, str := range macros {

			// True если содержит НЕ Латиницу ( https://github.com/google/re2/wiki/Syntax )
			//match, _ := regexp.MatchString(`\P{Latin}`, str)
			// True если содержит кирилицу
			match, _ := regexp.MatchString(`\p{Cyrillic}`, str)
			if match {
				return fmt.Errorf("'%s' - Макросы не нужно переводить.", str)
			}
		}

	}

	//Проверяем наличие переносов строки
	// if s.Contains(translate, "\n") {
	// 	return fmt.Errorf("Замените перенос строки на символ: \\n")
	// }

	return nil
}

func сheckErr(err error, text_opt ...string) {
	if err != nil {
		if len(text_opt) > 0 {
			fmt.Println(text_opt[0])
		}
		fmt.Println(err.Error())
		panic(err)
	}
}
