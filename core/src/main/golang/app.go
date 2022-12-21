package main

//#include "bridge.h"
import "C"

import (
	"unsafe"
)

func openRemoteContent(url string) (int, error) {
	u := C.CString(url)
	e := (*C.char)(C.malloc(1024))

	defer C.free(unsafe.Pointer(e))

	fd := C.open_content(u, e, 1024)

	return int(fd), nil
}

//export queryConfiguration
func queryConfiguration() *C.char {
	response := &struct{}{}

	return marshalJson(&response)
}
