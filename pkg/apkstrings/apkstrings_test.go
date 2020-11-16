package apkstrings

import (
	//"fmt"
	"os"
	"testing"
)

const (
	testxml = "strings.xml"
	outxml  = "out.xml"
)

const blob = `
<resources>
    <string name="abc_action_bar_home_description">Перейти на главный экран</string>
    <string name="abc_action_bar_up_description">Перейти вверх</string>
    <string name="abc_action_menu_overflow_description">Ещё</string>
    <string name="abc_action_mode_done">Готово</string>
    <string name="abc_activity_chooser_view_see_all">Показать все</string>
    <string name="abc_activitychooserview_choose_application">Выберите приложение</string>
    <string name="abc_capital_off">ВЫКЛ</string>
    <string name="abc_capital_on">ВКЛ</string>
</resources>`

func TestAll(t *testing.T) {

	infoInput, _ := os.Stat(testxml)

	prog := New(Config{
		Debug: os.Stdout,
	})

	dat, err := prog.LoadFile(testxml)
	if err != nil {
		t.Log(err)
	}

	i := 1
	for e := dat.Front(); e != nil; e = e.Next() {
		line := e.Value.([]string)
		t.Logf("[%d] %v", i, line)
		i++
	}

	t.Logf("[LoadFile] success, записей: %d", dat.Len())

	prog.SaveFile(outxml, dat)

	if infoOut, err := os.Stat(outxml); err == nil {
		if infoInput.Size() != infoOut.Size() {
			t.Error("[SaveFile] Входной и выходной фыйлы имеет разный размер!")
		}
		t.Logf("[SaveFile] XML create success, Size: %d", infoOut.Size())
	} else if os.IsNotExist(err) {
		t.Error("[SaveFile] XML crete failed, Файл не создался!")
	}

	//Убираем временные файлы
	os.Remove(outxml)

}
