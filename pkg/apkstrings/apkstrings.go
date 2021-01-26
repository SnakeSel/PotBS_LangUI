package apkstrings

import (
	"container/list"
	"encoding/xml"
	"io"
	"io/ioutil"
	"log"
	"os"
)

const moduleName = "apkstrings"

type Resources struct {
	XMLName xml.Name  `xml:"resources"`
	Strings []_string `xml:"string"`
}

type _string struct {
	Text string `xml:",chardata"`
	Name string `xml:"name,attr"`
}

// Config is the configuration struct you should pass to New().
type Config struct {
	// Debug is an optional writer which will be used for debug output.
	Debug io.Writer
}

type Translate struct {
	//err error

	log        *log.Logger
	conf       *Config
	sourceLang string
	targetLang string
	header     map[string]int
}

// New returns a new Translate.
func New(conf Config) *Translate {

	t := &Translate{conf: &conf}

	if conf.Debug == nil {
		conf.Debug = ioutil.Discard
	}

	t.header = map[string]int{
		"id":   0,
		"text": 1,
	}

	t.log = log.New(conf.Debug, "[apkstring]: ", log.LstdFlags)

	return t
}

type TData struct {
	Id   string
	Mode string
	Text string
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
	return moduleName
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

	id := t.GetHeaderNbyName("id")
	text := t.GetHeaderNbyName("text")

	Data := list.New()

	file, err := os.Open(filepach)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	// read our opened xmlFile as a byte array.
	byteValue, err := ioutil.ReadAll(file)
	if err != nil {
		return nil, err
	}
	// we initialize array
	var res Resources
	// we unmarshal our byteArray which contains our

	if err := xml.Unmarshal(byteValue, &res); err != nil {
		return nil, err
	}

	line := make([]string, t.GetHeaderLen())
	for i := 0; i < len(res.Strings); i++ {
		//fmt.Println("Name: " + res.String[i].Name)
		//fmt.Println("Value: " + res.String[i].Text)
		line[id] = res.Strings[i].Name
		line[text] = res.Strings[i].Text
		Data.PushBack(line)
		line = make([]string, t.GetHeaderLen())
	}

	return Data, nil
}

func (t *Translate) SaveFile(filepach string, datas *list.List) error {
	v := &Resources{}

	id := t.GetHeaderNbyName("id")
	text := t.GetHeaderNbyName("text")

	for e := datas.Front(); e != nil; e = e.Next() {
		line := e.Value.([]string)
		v.Strings = append(v.Strings, _string{Name: line[id], Text: line[text]})
	}
	xmlbyte, _ := xml.MarshalIndent(v, "", "    ")

	outfile, err := os.Create(filepach)
	if err != nil {
		t.log.Printf("Unable to create file: " + filepach)
		return err
	}
	defer outfile.Close()

	outfile.WriteString("<?xml version=\"1.0\" encoding=\"utf-8\"?>\n")
	_, err = outfile.Write(xmlbyte)
	if err != nil {
		t.log.Printf("Unable write xml to " + filepach)
		return err
	}
	outfile.WriteString("\n")

	return nil

}

func (t *Translate) ValidateTranslate(sourseText, targetText string) []error {
	return nil
}
