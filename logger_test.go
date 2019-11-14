package logger

import (
	"fmt"
	"testing"
	"time"
)

func testLevel(level int, file, path string, filetype int) {
	InitLogging(filetype, file, path, 5, level)
	format, val := "haha %v", "yzh"
	Debugf(format, val)
	Infof(format, val)
	Warningf(format, val)
	Errorf(format, val)

	Debugln(val)
	Infoln(val)
	Warningln(val)
	Errorln(val)
	CloseDefault()
	time.Sleep(time.Second)
}
func TestLogger(t *testing.T) {
	fmt.Println("test debug")
	testLevel(DEBUG, "", "", STDOUT)

	fmt.Println("test info")
	testLevel(INFO, "", "", STDOUT)

	fmt.Println("test warning")
	testLevel(WARNING, "", "", STDOUT)

	fmt.Println("test error")
	testLevel(ERROR, "", "", STDOUT)

	fmt.Println("test file")
	testLevel(DEBUG, "log.log", "./", FILE)
}
