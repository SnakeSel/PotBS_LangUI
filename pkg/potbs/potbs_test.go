package potbs

import (
	//"fmt"
	"os"
	"testing"
)

const (
	testdatfile = "test.dat"
	testFailDat = "test_fail.dat"
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

func TestErroLoad(t *testing.T) {
	prog := New(Config{
		Debug: os.Stdout,
	})
	_, err := prog.LoadFile(testFailDat)
	if err == nil {
		t.Error("[LoadErrDat] failed: Failed dat file load success.")
	}
}

func TestAll(t *testing.T) {

	prog := New(Config{
		//Debug: os.Stdout,
	})
	dat, err := prog.LoadFile(testdatfile)
	if err != nil {
		t.Log(err)
	}

	if dat.Len() != datlen {
		i := 1
		for e := dat.Front(); e != nil; e = e.Next() {
			line := e.Value.([]string)
			t.Logf("[%d] %v", i, line)
			i++
		}
		t.Errorf("[LoadFile] failed: Должно быть: %d записей, распарсено: %d", datlen, dat.Len())
	}

	//id := prog.Header["id"]
	text := prog.GetHeaderNbyName("text")

	i := 0
	for e := dat.Front(); e != nil; e = e.Next() {
		line := e.Value.([]string)
		// Проверка строки на длинну
		if val, ok := checkLen[i]; ok {
			if val != len(line[text]) {
				t.Errorf("[LoadFile] failed: длинна записи %d = %d, вместо %d", i, len(line[text]), val)
			}
		}
		i++
	}

	t.Logf("[LoadFile] success, записей: %d", dat.Len())
	t.Log("\n\n**************** Test Save File *************************")

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
