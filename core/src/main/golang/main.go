package main

//#include "bridge.h"
import "C"

import (
	"cfa/config"
	"cfa/delegate"
	"cfa/tunnel"
)

func main() {
	panic("Stub!")
}

//export coreInit
func coreInit(home C.c_string) {
	h := C.GoString(home)
	delegate.Init(h)
	reset()
}

//export reset
func reset() {
	config.LoadDefault()
	tunnel.ResetStatistic()
	tunnel.CloseAllConnections()
}
