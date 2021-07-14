module github.com/slok/reload/_examples/simpleapp

go 1.16

require (
	github.com/alecthomas/template v0.0.0-20190718012654-fb15b899a751 // indirect
	github.com/alecthomas/units v0.0.0-20210208195552-ff826a37aa15 // indirect
	github.com/fsnotify/fsnotify v1.4.9
	github.com/oklog/run v1.1.0
	github.com/sirupsen/logrus v1.8.1
	github.com/slok/reload v0.0.0
	golang.org/x/sys v0.0.0-20210630005230-0f9fa26af87c // indirect
	gopkg.in/alecthomas/kingpin.v2 v2.2.6
)

replace github.com/slok/reload => ../../
