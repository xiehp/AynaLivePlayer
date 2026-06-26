#$vlc_sdk = "D:\work\libs\VideoLAN.LibVLC.Windows.3.0.21\build\x64"
#$mpv_path = (scoop prefix mpv)
#$env:CGO_CFLAGS = "-I$mpv_path\include -I$vlc_sdk\include"
#$env:CGO_LDFLAGS = "-L$mpv_path -L$vlc_sdk"

$vlc_sdk = "D:\work\env\vlc"
$mpv_path = "D:\work\env\mpv"
$env:CGO_CFLAGS = "-I$mpv_path\include  -I$vlc_sdk\include"
$env:CGO_LDFLAGS = "-L$mpv_path  -L$vlc_sdk"

# ... more setup, see makefile
# go build -o AynaLivePlayer.exe -ldflags -H=windowsgui app/main.go

$env:Path = "$mpv_path;$vlc_sdk;$env:Path"
# 4. 编译
go run app/main.go --dev

pause