package main

//#include "bridge.h"
import "C"

import (
	"runtime"

	"cfa/native/config"
	"cfa/native/delegate"
	"cfa/native/tunnel"
)

func main() {
	panic("Stub!")
}

//export coreInit
func coreInit(home, versionName C.c_string, sdkVersion C.int) {
	h := C.GoString(home)
	v := C.GoString(versionName)
	s := int(sdkVersion)

	delegate.Init(h, v, s)

	reset()
}

//export reset
func reset() {
	config.LoadDefault()
	tunnel.ResetStatistic()
	tunnel.CloseAllConnections()
}

//export forceGc
func forceGc() {
	go func() {
		runtime.GC()
	}()
}
