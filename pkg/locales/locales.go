// locale project locale.go
package locales

import (
	//"fmt"

	lang "golang.org/x/text/language"
	msg "golang.org/x/text/message"
	"gopkg.in/ini.v1"
)

type Printer struct {
	*msg.Printer
}

// New returns a new printer.
func New(file, lg_name string) (*Printer, error) {
	lg, err := lang.Parse(lg_name)
	if err != nil {
		return nil, err
	}
	p := msg.NewPrinter(lg)

	loadFromIni(file)

	return &Printer{p}, nil
}

func loadFromIni(file string) {
	cfg, _ := ini.LooseLoad(file)
	// if err != nil {
	// 	//
	// }
	//fmt.Println(cfg.SectionStrings())
	for _, section := range cfg.Sections() {
		if section.Name() == "DEFAULT" {
			continue
		}
		lg, _ := lang.Parse(section.Name())
		for _, key := range section.Keys() {
			msg.SetString(lg, key.Name(), key.Value())

		}

	}

}

// Получаем карту [Секция]LocaleName из файла перевода
// Например: [en-US]English
func GetAllLocaleName(file string) map[string]string {
	locales := make(map[string]string)

	locIni, _ := ini.LooseLoad(file)
	// if err != nil {
	// 	//
	// }
	for _, section := range locIni.Sections() {
		if section.Name() == "DEFAULT" {
			continue
		}
		LocaleName, err := section.GetKey("LocaleName")
		if err != nil {
			continue
		}

		locales[section.Name()] = LocaleName.Value()

	}
	return locales
}
