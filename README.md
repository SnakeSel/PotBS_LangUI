# PotBS_LangUI

[English version](README_EN.md) 🇬🇧

Графическая программа для перевода языковых файлов SOE T4.  
T4 используется в играх:
* Корсары Онлайн: Pirates of the burning sea
* EverQuest 2
* PlanetSide 2
* H1Z1
* ...

Написана на [Go](https://golang.org/) с использованием [gotk3](https://github.com/gotk3/gotk3)
* При сохранении DAT файла перевода формирует для него DIR файл.
* Поддерживает шаблоны переводов.
* Поддержка Google Translate.
* Поддержка проверки перевода (Макрос закрыт, макрос не переведен...)
![](screen/main.png)
## Установка и запуск
Скачать [актуальный релиз](https://github.com/SnakeSel/PotBS_LangUI/releases)  
Распаковать архив и запустить:
- Windows: `potbs_langui.exe`
- Linux: `./potbs_langui`

## Сборка из исходников
#### Необходимо установить пакеты разработки GTK3:
- Windows: [msys2](https://www.gtk.org/docs/installations/windows/#using-gtk-from-msys2-packages) или [Chocolatey](https://github.com/gotk3/gotk3/wiki/Installing-on-Windows)
- [Linux](https://github.com/gotk3/gotk3/wiki/Installing-on-Linux)

#### Загружаем исходный код PotBS_LangUI:
```sh
$ git clone https://github.com/SnakeSel/PotBS_LangUI
```
#### Переходим в директорию PotBS_LangUI:
```sh
$ cd PotBS_LangUI
```
#### Загружаем зависимости
```sh
go mod tidy
```
#### Компилируем:
- Windows:
```sh
go build -ldflags "-H=windowsgui -s -w"
```
- Linux:
```sh
go build
```
#### Запускаем:
```sh
./potbs_langui
```
