# gnome-background-changer

Change background randomly via app indicator

Build with
```
go build
```

This is a cgo build, so it needs a C compiler and the appindicator headers. On
Debian/Ubuntu that is
```
sudo apt install build-essential libayatana-appindicator3-dev
```
The compiler is not optional and its absence is not obvious: with no C compiler
in `PATH` the Go toolchain silently defaults `CGO_ENABLED` to 0, which drops
every file that implements systray on Linux, and the build fails with
`undefined: nativeLoop` and friends rather than with anything mentioning cgo.

On systems without the new `ayatana` appindicator (e.g. Fedora), use
```
go build -tags=legacy_appindicator
```

Create a configfile at `~/.config/bgchanger/config.json` with content similar to
the following:
```
{
    "pictures_dir": "/home/you/Pictures/Backgrounds",
    "interval": "15m"
}
```
