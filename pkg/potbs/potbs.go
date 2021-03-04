package potbs

import (
	"bufio"
	"bytes"
	"fmt"
	"os"

	str "strings"

	"container/list"
	"io"
	"io/ioutil"
	"log"
	"path/filepath"
	"regexp"
	"strconv"
)

var BeginByte = []byte("ï»¿")

// Config is the configuration struct you should pass to New().
type Config struct {
	// Debug is an optional writer which will be used for debug output.
	Debug io.Writer
}

type tDir struct {
	Id     int
	pos    int
	lenght int
}

type Translate struct {
	//err error

	log            *log.Logger
	conf           *Config
	sourceLang     string
	targetLang     string
	header         map[string]int
	moduleName     string
	validateChecks map[string]bool // вкл\откл проверок
}

// New returns a new Translate.
func New(conf Config) *Translate {

	t := &Translate{conf: &conf}

	if conf.Debug == nil {
		conf.Debug = ioutil.Discard
	}

	t.header = map[string]int{
		"id":   0,
		"mode": 1,
		"text": 2,
	}

	t.log = log.New(conf.Debug, "[potbs]: ", log.LstdFlags)

	t.moduleName = "potbs"

	t.validateChecks = map[string]bool{
		"Numeric":     false,
		"MacrosNotRu": false,
		//"MacrosEND":   false,
		"comparingMacros": false,
	}

	return t
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
func indexN(strg, substr string, n int) int {

	ind := 0
	pos := 0
	for i := 0; i < n; i++ {
		ind = str.Index(strg[pos:], substr)
		if ind == -1 {
			return ind
		}
		pos += ind
		pos += len(substr)
	}

	return pos
}

func checkModeLine(line string) string {
	switch {
	case str.Contains(line, "ucdt"):
		return "ucdt"
	case str.Contains(line, "ucdn"):
		return "ucdn"
	case str.Contains(line, "mcdt"):
		return "mcdt"
	case str.Contains(line, "mcdn"):
		return "mcdn"
	default:
		return ""
	}

	return ""

}

func (t *Translate) SetSourceLang(lang string) {
	t.sourceLang = lang
}

func (t *Translate) GetSourceLang() string {
	return t.sourceLang
}

func (t *Translate) SetTargetLang(lang string) {
	t.targetLang = lang
}
func (t *Translate) GetTargetLang() string {
	return t.targetLang

}

func (t *Translate) GetHeaderLen() int {
	return len(t.header)
}
func (t *Translate) GetHeader() map[string]int {
	return t.header
}

func (t *Translate) GetModuleName() string {
	return t.moduleName
}

func (t *Translate) GetChecks() map[string]bool {
	return t.validateChecks
}

func (t *Translate) SetCheckActivebyName(name string, enable bool) error {
	if _, ok := t.validateChecks[name]; ok {
		t.validateChecks[name] = enable
		return nil
	} else {
		return fmt.Errorf("Check %s not found", name)
	}

	return nil
}

func (t *Translate) GetCheckDescriptionbyName(name string) (string, error) {
	switch name {
	case "Numeric":
		return "the sum of the numbers on both sides is the same", nil
	case "MacrosNotRu":
		return "macros are not translated into Russian", nil
	case "MacrosEND":
		return "macros closed", nil
	case "comparingMacros":
		return "translation macros are the same as in the original", nil
	default:
		return "", fmt.Errorf("Check %s not found", name)
	}
	return "", fmt.Errorf("Check %s not found", name)
}

// Возвращает номер заголовка с указанным именем.
// Если такого имени нет, вернет -1
func (t *Translate) GetHeaderNbyName(name string) int {
	if val, ok := t.header[name]; ok {
		return val
	} else {
		return -1
	}

}

func (t *Translate) LoadFile(filepach string) (*list.List, error) {

	// Входной dat файл со списком строк вида:
	// <id>\t<вид строки>\t<строка>\r\n
	// вид строки:
	// ucdt - текст. подсчитваем всю строку
	// ucdn - пустая строка
	// mcdt - Текст со скриптом. Далее строка имеет вид: <текст>\t<scriptID>\t<script name>. Подсчитваем только <текст>
	// mcdn - Пустая строка со скриптом
	// ugdt - названия предметов?

	id := t.GetHeaderNbyName("id")
	text := t.GetHeaderNbyName("text")
	mode := t.GetHeaderNbyName("mode")

	Data := list.New()

	file, err := os.Open(filepach)
	if err != nil {
		return list.New(), err
	}
	defer file.Close()

	// Проверка на соответствие файла
	firstbit := make([]byte, len(BeginByte))

	_, err = file.Read(firstbit)
	if err != nil {
		return nil, err
	}
	// t.log.Println(string(BeginByte))
	// t.log.Println(BeginByte)

	// t.log.Println(string(firstbit))
	// t.log.Println(firstbit)

	if bytes.Compare(firstbit, BeginByte) != 0 {
		return nil, fmt.Errorf("Error Load file %s. The file does not start with %s", filepach, string(BeginByte))
	}

	//Сканируем файл
	scanner := bufio.NewScanner(file)

	//var line []string
	line := make([]string, 3)
	var lineLen int
	var splitline []string

	// Концом строки считается \r\n
	scanner.Split(scanCRLF)
	lineN := 0
	first := true

	for scanner.Scan() {

		// Подсчитваем длину и разбиваем строку по "\t"

		// Проверка не искользуется т.к. заголовок уже считали. Потом убрать
		// if first {
		// 	// Первая строка сожержит заголовок.
		// 	// Пока просто отбрасываем 6 байт
		// 	lineall := scanner.Text()[6:]
		// 	splitline = str.Split(lineall, "\t")
		// 	lineLen = len(lineall)
		// 	t.log.Printf("[%d] Len: %d\t(%v)", lineN, lineLen, splitline)
		// } else {
		splitline = str.SplitN(scanner.Text(), "\t", 3)
		lineLen = len(scanner.Bytes())
		//lineLen = utf8.RuneCount(scanner.Bytes())
		t.log.Printf("[%d] Len: %d\t(%v)", lineN, lineLen, splitline)
		// }

		// костыль пустой строки
		if lineLen == 0 {
			t.log.Printf("[%d] %s\t(%s)", lineN, "пустая строка, пропускаем", scanner.Text())
			continue
		}

		// Если вдруг вместо \t сделали разделение пробелами
		if len(splitline) != 3 {
			chmode := checkModeLine(scanner.Text())
			if chmode != "" {
				t.log.Printf("Длинна строки %d, но содержит: %s", len(splitline), chmode)
				tmpsplitline := str.SplitN(scanner.Text(), chmode, 2)
				splitline2 := make([]string, 3)
				//t.log.Println(tmpsplitline)
				splitline2[0] = str.TrimSpace(tmpsplitline[0])
				splitline2[1] = chmode
				splitline2[2] = tmpsplitline[1]
				splitline = splitline2
				//t.log.Println(splitline)
				t.log.Printf("New splitline len: %d", len(splitline))
			}

		}

		// Определяем появление нового ID.
		if len(splitline) >= 2 && len(splitline[1]) == 4 {
			//t.log.Printf("check ID: len %d, mode: %s", len(splitline), splitline[1])
			if _, ok := strconv.Atoi(splitline[0]); ok == nil {
				//если только начали, то парсим дальше, если нет, заносим распарсенное
				if first {
					line[id] = splitline[0]
					line[text] = ""
					first = false
				} else {
					t.log.Printf("[%d] EOF id:(%s)", lineN, line[id])
					Data.PushBack(line)
					line = make([]string, 3)
					lineN += 1
					line[id] = splitline[0]
					line[text] = ""

				}
				t.log.Printf("[%d] Start\t id:(%s)", lineN, line[id])
			}
		}

		// Проверяем кол-во разделенных элементов. Должно быть 3

		switch {
		case len(splitline) >= 3:
			t.log.Println("mode 1 (len3)")
			line[mode] = splitline[1]
			line[text] += splitline[2]
		case len(splitline) == 2:
			// 2 быват при пустой строке (ucdn)
			t.log.Println("mode 2 (len2)")
			if len(splitline[1]) == 4 {
				line[mode] = splitline[1]
				line[text] = ""
			} else {
				line[text] += "\n" + scanner.Text()
			}
		default:
			// При строке с \r\n в середине
			t.log.Println("mode 3")
			//line.Mode = "none"
			line[text] += "\n" + scanner.Text()
		}

	}

	// Заносим последнюю строку
	Data.PushBack(line)

	return Data, nil
}

func (t *Translate) SaveFile(filepach string, datas *list.List) error {

	id := t.GetHeaderNbyName("id")
	text := t.GetHeaderNbyName("text")
	mode := t.GetHeaderNbyName("mode")

	dirs := make([]tDir, 0)
	var dir tDir
	var linelen int
	pos := 0 // Бит в файле

	filedat, err := os.Create(filepach)
	if err != nil {
		t.log.Printf("Unable to create file: " + filepach)
		return err
	}
	defer filedat.Close()

	// Iterate through list and print its contents.
	first := true
	for e := datas.Front(); e != nil; e = e.Next() {
		line := e.Value.([]string)
		t.log.Println(line)
		if first {
			filedat.WriteString(fmt.Sprintf("%s%s\t%s\t%s\r\n", BeginByte, line[id], line[mode], line[text]))
			pos += 6 //BeginByte
			first = false
		} else {
			filedat.WriteString(fmt.Sprintf("%s\t%s\t%s\r\n", line[id], line[mode], line[text]))
		}

		dir.Id, _ = strconv.Atoi(line[id])
		dir.pos = pos

		// Расчитываем длину строки
		linelen = len(line[id])
		linelen += 1 //\t
		linelen += len(line[mode])
		linelen += 1 //\t
		mcdtlen := linelen
		linelen += len(line[text])
		// Ебала с размером. В позицию идет вся длинна (linelen), а в размер только длина текста (mcdtlen).
		switch line[mode] {
		case "mcdt":
			// mcdt - Текст со скриптом. Далее строка имеет вид: <текст>\t<scriptID>\t<script name>. Подсчитваем только <текст>
			ind := str.Index(line[text], "\t")
			// -1 - не найдено
			if ind == -1 {
				mcdtlen = linelen
			} else {
				mcdtlen += len(line[text][:ind])
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

	//Создаем dir файл
	patch, file := filepath.Split(filepach)
	filedir, err := os.Create(patch + str.TrimSuffix(file, filepath.Ext(file)) + ".dir")
	if err != nil {
		t.log.Println("Error save DIR file")
		return err
	}
	defer filedir.Close()

	// Записываем начало
	filedir.WriteString(fmt.Sprintf("## Count:\t%d\r\n", len(dirs)))
	filedir.WriteString("## Game:\tPBS\r\n")

	locale := langName(t.GetTargetLang())
	if locale != "" {
		filedir.WriteString(fmt.Sprintf("## Locale:\t%s\r\n", locale))
	}

	for _, dir := range dirs {
		filedir.WriteString(fmt.Sprintf("%d\t%d\t%d\td\r\n", dir.Id, dir.pos, dir.lenght))
	}

	return nil
}

// func IsRusByUnicode(str string) bool {
// 	for _, r := range str {
// 		if unicode.Is(unicode.Cyrillic, r) {
// 			return true
// 		}
// 	}
// 	return false
// }

func (t *Translate) ValidateTranslate(sourceText, targetText string) []error {

	var validateErr []error = nil

	//Проверяем перевод макросов на Русский
	if t.validateChecks["MacrosNotRu"] {
		re_macros := regexp.MustCompile(`\[\!(.+?)\!\]`)
		macros := re_macros.FindAllString(targetText, -1)
		if len(macros) != 0 {
			for _, str := range macros {

				// True если содержит НЕ Латиницу ( https://github.com/google/re2/wiki/Syntax )
				//match, _ := regexp.MatchString(`\P{Latin}`, str)
				// True если содержит кирилицу
				match, _ := regexp.MatchString(`\p{Cyrillic}`, str)
				if match {
					validateErr = append(validateErr, fmt.Errorf("'%s' - Макросы не нужно переводить.", str))
				}
			}

		}
	}
	//Проверяем наличие переносов строки
	// if s.Contains(targetText, "\n") {
	// 	return fmt.Errorf("Замените перенос строки на символ: \\n")
	// }

	// Если какой - то из переводов пуст, дальнейшие проверки безсмысленны
	if sourceText == "" || targetText == "" {
		return validateErr
	}

	// Проверяем на окончание и убираем макросы
	noMacSource, err := removeMacros(sourceText)
	if err != nil {
		//validateErr = append(validateErr, err)
	}

	noMacTarget, err := removeMacros(targetText)
	if err != nil {
		validateErr = append(validateErr, err)
	}

	// проверка соответсвия цифровых значений (например сила удара и т.д.)
	if t.validateChecks["Numeric"] {
		if noMacSource != "" && noMacTarget != "" {
			err := validateNumeric(noMacSource, noMacTarget)
			if err != nil {
				validateErr = append(validateErr, err)
			}
		}
	}

	// сравнение макросов
	if t.validateChecks["comparingMacros"] {
		re_macros := regexp.MustCompile(`\[\!(.+?)\!\]`)
		macrosSource := re_macros.FindAllString(sourceText, -1)
		macrosTarget := re_macros.FindAllString(targetText, -1)
		var found bool
		if len(macrosTarget) != 0 {
			// Идем по всем макросам перевода
			for _, macros := range macrosTarget {
				found = false
				// сверяем с макросами оригинала
				for ind, m := range macrosSource {
					// Если макросы совпдают, убираем его из исходных
					if macros == m {
						found = true
						macrosSource = removeIndex(macrosSource, ind)
						break
					}
				}
				// Если не нашли, выводим сообщение
				if !found {
					validateErr = append(validateErr, fmt.Errorf("Макрос %s не найден в оригинале.", macros))
				}

			}

			// Если после проверки остались значения в macrosSource, значит они не использовались в переводе
			if len(macrosSource) != 0 {
				for _, macros := range macrosSource {
					validateErr = append(validateErr, fmt.Errorf("Макрос %s не найден в переводе.", macros))
				}
			}
		}
	}
	return validateErr
}

func removeIndex(s []string, index int) []string {
	return append(s[:index], s[index+1:]...)
}

// проверка соответсвия цифровых значений (например сила удара и т.д.)
func validateNumeric(sourceText, targetText string) error {

	numSource := getNumericText(sourceText)
	numTarget := getNumericText(targetText)

	if numSource != "" && numTarget != "" {
		// Чтобы не зависеть от перемены мест, берем сумму
		if getSum(numSource) != getSum(numTarget) {
			return fmt.Errorf("Check numerical value! %s != %s", numSource, numTarget)
		}
	}
	return nil
}

func getNumericText(text string) string {
	var out string
	for _, char := range text {
		switch string(char) {
		case "1", "2", "3", "4", "5", "6", "7", "8", "9", "0":
			out += string(char)
		}

	}
	return out
}

func getSum(text string) int {

	if text == "" {
		return 0
	}

	var out int = 0

	for _, char := range text {
		i, err := strconv.Atoi(string(char))
		if err == nil {
			out += i
		}
	}
	return out
}

func removeMacros(text string) (string, error) {
	// Проверка [!*!]
	openIndx := str.Index(text, "[!")
	//Пока мы находим в тексте начало макроса
	for openIndx != -1 {
		endIndx := str.Index(text, "!]")
		switch {
		case endIndx == -1:
			return "", fmt.Errorf("Not found END macros: ( !] )")
		case endIndx < openIndx:
			return "", fmt.Errorf("Not found START macros: ( [! )")

		}
		text = text[:openIndx] + text[endIndx+2:]
		openIndx = str.Index(text, "[!")
	}

	// Проверка [:*:]
	openIndx = str.Index(text, "[:")
	//Пока мы находим в тексте начало макроса
	for openIndx != -1 {
		lastIndx := str.Index(text, ":]")
		switch {
		case lastIndx == -1:
			return "", fmt.Errorf("Not found END macros: ( :] )")
		case lastIndx < openIndx:
			return "", fmt.Errorf("Not found START macros: ( [: )")

		}
		text = text[:openIndx] + text[lastIndx+2:]
		openIndx = str.Index(text, "[:")
	}

	return text, nil
}

func langName(lang string) string {
	switch str.ToUpper(lang) {
	case "RU":
		return "ru_RU"
	case "EN":
		return "en_US"
	case "FR":
		return "fr_FR"
	case "DE":
		return "de_DE"
	case "ES":
		return "es_ES"
	default:
		return ""
	}

	return ""
}
