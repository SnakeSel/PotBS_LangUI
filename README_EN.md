# PotBS_LangUI

[Русская версия](README.md) 🇷🇺

Graphic program for translating language files SOE T4.  
T4 is used in games:
* Pirates of the burning sea
* EverQuest 2
* PlanetSide 2
* H1Z1
* ...

Written in [Go](https://golang.org/) using [gotk3](https://github.com/gotk3/gotk3)
* When saving the DAT file, the translation generates a DIR file for it..
* Supports translation templates.
* Support for Google Translate.
* Support for translation checking.
![](screen/main.png)
## Install and run
Download [release](https://github.com/SnakeSel/PotBS_LangUI/releases)  
Unpack and run:
- Windows: `potbs_langui.exe`
- Linux: `./potbs_langui`

## Build from sourse
#### You need to install GTK3 development packages:
- Windows: [msys2](https://www.gtk.org/docs/installations/windows/#using-gtk-from-msys2-packages) or [Chocolatey](https://github.com/gotk3/gotk3/wiki/Installing-on-Windows)
- [Linux](https://github.com/gotk3/gotk3/wiki/Installing-on-Linux)

#### Download PotBS_LangUI:
```sh
$ go get github.com/snakesel/potbs_langui/
```
#### Go to the PotBS_LangUI directory:
```sh
$ cd $GOPATH/src/github.com/snakesel/potbs_langui/
```
#### Build:
- Windows:
```sh
go build -ldflags "-H=windowsgui -s -w"
```
- Linux:
```sh
go build
```
#### Running:
```sh
./potbs_langui
```
