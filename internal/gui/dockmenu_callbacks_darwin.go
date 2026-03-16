//go:build darwin

package gui

import "C"
import "unsafe"

// menuSnapshot holds the window list captured by goGetWindowCount so that
// subsequent goGetWindowTitle/goGetWindowID calls index into the same slice.
// All three are called sequentially on the main thread during dock menu build.
var menuSnapshot []WindowEntry

//export goGetWindowCount
func goGetWindowCount() C.int {
	if activeHost == nil {
		menuSnapshot = nil
		return 0
	}
	menuSnapshot = activeHost.WindowList()
	return C.int(len(menuSnapshot))
}

//export goGetWindowID
func goGetWindowID(index C.int) *C.char {
	i := int(index)
	if i < 0 || i >= len(menuSnapshot) {
		return nil
	}
	return C.CString(menuSnapshot[i].ID)
}

//export goGetWindowTitle
func goGetWindowTitle(index C.int) *C.char {
	i := int(index)
	if i < 0 || i >= len(menuSnapshot) {
		return nil
	}
	return C.CString(menuSnapshot[i].Filename)
}

//export goDockMenuActivate
func goDockMenuActivate(windowID *C.char) {
	if activeHost == nil {
		return
	}
	id := C.GoString(windowID)
	activeHost.primaryWV.Dispatch(func() {
		activeHost.ActivateWindow(id)
	})
}

//export goDockMenuClose
func goDockMenuClose(windowID *C.char) {
	if activeHost == nil {
		return
	}
	id := C.GoString(windowID)
	activeHost.primaryWV.Dispatch(func() {
		activeHost.CloseWindow(id)
	})
}

//export goDockMenuOpenFile
func goDockMenuOpenFile(path *C.char) {
	if activeHost == nil {
		return
	}
	p := C.GoString(path)
	go activeHost.OpenFile(p)
}

// Ensure unsafe is used (required for //export files).
var _ = unsafe.Pointer(nil)
