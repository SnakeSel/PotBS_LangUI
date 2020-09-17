package potbs

import (
	//"fmt"
	"os"
	"testing"
)

const (
	testdatfile = "test.dat"
	datlen      = 15 //кол-во записей в тестовом файле

	testoutdat = "testout.dat"
	testoutdir = "testout.dir"

	datSize = 1692 // Размер создаваемого файла
	dirSize = 321  // Размер создаваемого файла
)

var checkLen = map[int]int{
	0:  0,
	4:  215,
	6:  190,
	12: 429,
}

func TestAll(t *testing.T) {

	prog, _ := New(Config{
		Debug: os.Stdout,
	})
	dat, err := prog.LoadFile(testdatfile)
	if err != nil {
		t.Log(err)
	}
	//fmt.Print(len(dat))
	if len(dat) != datlen {
		for N, line := range dat {
			t.Logf("[%d] %v", N, line)
		}
		t.Errorf("[LoadFile] failed: Должно быть: %d записей, распарсено: %d", datlen, len(dat))
	}

	for N, line := range dat {
		// Проверка строки на длинну
		if val, ok := checkLen[N]; ok {
			if val != len(line.Text) {
				t.Errorf("[LoadFile] failed: длинна записи %d = %d, вместо %d", N, len(line.Text), val)
			}
		}
	}

	t.Logf("[LoadFile] success, записей: %d", len(dat))

	prog.SaveFile(testoutdat, dat)
	if info, err := os.Stat(testoutdat); err == nil {
		t.Logf("[SaveFile] DAT create success, Size: %d", info.Size())
		if datSize != info.Size() {
			t.Error("[SaveFile] DAT файл имеет неправильный размер!")
		}
	} else if os.IsNotExist(err) {
		t.Error("[SaveFile] DAT crete failed, Файл не создался!")
	}

	if info, err := os.Stat(testoutdir); err == nil {
		t.Logf("[SaveFile] DIR create success, Size: %d", info.Size())
		if dirSize != info.Size() {
			t.Error("[SaveFile] DIR файл имеет неправильный размер!")
		}
	} else if os.IsNotExist(err) {
		t.Error("[SaveFile] DIR crete failed, Файл не создался!")
	}

	//Убираем временные файлы
	os.Remove(testoutdat)
	os.Remove(testoutdir)

}
