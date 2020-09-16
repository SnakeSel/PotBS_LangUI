@echo off
clear

set proj=PotBS_LangUI
set today=%date:~6,4%%date:~3,2%%date:~0,2%
set mingw=C:\msys64\mingw64
set sevenz="C:\Program Files\7-Zip\7z.exe"
set builddir=%CD%\Build\%today%\%proj%
set libdir=%builddir%
set libs=libatk-1.0-0.dll libbz2-1.dll libcairo-2.dll libcairo-gobject-2.dll libepoxy-0.dll libexpat-1.dll libffi-6.dll libfontconfig-1.dll libfreetype-6.dll libgcc_s_seh-1.dll libgdk-3-0.dll libgdk_pixbuf-2.0-0.dll libgio-2.0-0.dll libgit2.dll libglib-2.0-0.dll libgmodule-2.0-0.dll libgobject-2.0-0.dll libgraphite2.dll libgtk-3-0.dll libharfbuzz-0.dll libiconv-2.dll libintl-8.dll libpango-1.0-0.dll libpangocairo-1.0-0.dll libpangoft2-1.0-0.dll libpangowin32-1.0-0.dll libpcre-1.dll libpixman-1-0.dll libpng16-16.dll libstdc++-6.dll libwinpthread-1.dll zlib1.dll libfribidi-0.dll libthai-0.dll libdatrie-1.dll
rem �����: libjasper-4.dll libjpeg-8.dll

echo Building ...
go build -ldflags "-H=windowsgui -s -w"
rem go build

if errorlevel 0 ( 
	echo Build OK 
	) else (
	echo "ERROR build"
	@pause 
	exit /b 1
)

echo Copy libs ...
if not exist %libdir% (
	 md %libdir%
)

for  %%l in (%libs%) do (
	xcopy %mingw%\bin\%%l %libdir%
rem	echo errorlevel %errorlevel%
rem	if errorlevel 0 ( 
rem		echo "%%l copy OK" 
rem	) else (
rem		echo "ERROR copy %%l"
rem	)
)


echo Copy Data ...
xcopy potbs_langui.exe %builddir%
xcopy data %builddir%\data\ /S


echo Create etc\gtk-3.0\settings.ini ...
set confdir=%builddir%\etc\gtk-3.0
if not exist %confdir% (
	md %confdir%
)
echo [Settings] > %confdir%\settings.ini
echo gtk-theme-name=Windows10 >> %confdir%\settings.ini
echo gtk-font-name=Segoe UI 9 >> %confdir%\settings.ini


echo Copy pixbuf
xcopy %mingw%\lib\gdk-pixbuf-2.0 %builddir%\lib\gdk-pixbuf-2.0\ /S


echo Copy Adwaita ...
set adwaita=%mingw%\share\icons\Adwaita
set adwaita_build=%builddir%\share\icons\Adwaita

rem for  %%r in (16x16,22x22,24x24,32x32,48x48,64x64,96x96,256x256) do (
for  %%r in (16x16,22x22,24x24,32x32,48x48) do (
rem	md %adwaita_build%\%%r\devices
rem	md %adwaita_build%\%%r\actions
	md %adwaita_build%\%%r\legacy

rem	xcopy %adwaita%\%%r\legacy\media-floppy.png %adwaita_build%\%%r\devices\
rem	xcopy %adwaita%\%%r\devices\media-floppy-symbolic.symbolic.png %adwaita_build%\%%r\devices\
	xcopy %adwaita%\%%r\legacy\media-floppy.png %adwaita_build%\%%r\legacy\

rem	xcopy %adwaita%\%%r\legacy\tools-check-spelling.png %adwaita_build%\%%r\actions\
	xcopy %adwaita%\%%r\legacy\tools-check-spelling.png %adwaita_build%\%%r\legacy\

)

(echo [Icon Theme]
echo Name=Adwaita
echo Comment=The Only One
echo Example=folder
echo. 
echo # KDE Specific Stuff
echo DisplayDepth=32
echo LinkOverlay=link_overlay
echo LockOverlay=lock_overlay
echo ZipOverlay=zip_overlay
echo DesktopDefault=48
echo DesktopSizes=16,22,32,48,64,72,96,128
echo ToolbarDefault=22
echo ToolbarSizes=16,22,32,48
echo MainToolbarDefault=22
echo MainToolbarSizes=16,22,32,48
echo SmallDefault=16
echo SmallSizes=16
echo PanelDefault=32
echo PanelSizes=16,22,32,48,64,72,96,128
echo. 
echo # Directory list
echo Directories=16x16/devices,16x16/legacy,22x22/devices,22x22/legacy,24x24/devices,24x24/legacy,32x32/devices,32x32/legacy,48x48/devices,48x48/legacy,
echo. 
echo [16x16/devices]
echo Context=Devices
echo Size=16
echo Type=Fixed
echo. 
echo [16x16/legacy]
echo Context=Legacy
echo Size=16
echo Type=Fixed
echo. 
echo [22x22/devices]
echo Context=Devices
echo Size=22
echo Type=Fixed
echo. 
echo [22x22/legacy]
echo Context=Legacy
echo Size=22
echo Type=Fixed
echo. 
echo [24x24/devices]
echo Context=Devices
echo Size=24
echo Type=Fixed
echo. 
echo [24x24/legacy]
echo Context=Legacy
echo Size=24
echo Type=Fixed
echo. 
echo [32x32/devices]
echo Context=Devices
echo Size=32
echo Type=Fixed
echo. 
echo [32x32/legacy]
echo Context=Legacy
echo Size=32
echo Type=Fixed
echo. 
echo [48x48/devices]
echo Context=Devices
echo Size=48
echo Type=Fixed
echo. 
echo [48x48/legacy]
echo Context=Legacy
echo Size=48
echo Type=Fixed
) > %adwaita_build%\index.theme


echo Copy hicolor ...


echo Copy Win10 themas ...
%sevenz% x "%CD%\pkg\gtk-3.20.7z" -o"%builddir%\share\themes\Windows10\gtk-3.0\"


echo glib-compile-schemas ...
md %builddir%\share\glib-2.0\schemas
cd %builddir%
glib-compile-schemas share/glib-2.0/schemas

echo Create Archive
cd %builddir%\..\
%sevenz% a "%builddir%\..\..\%proj%_Win64_%today%.7z" "%proj%"

@pause
exit /b 0