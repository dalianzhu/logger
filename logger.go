// Package logger 是系统日志的封装，主要在之上封装了Error，Info两个函数。并提供了跨日期
// 自动分割日志文件的功能。
// 可以在InitLogging 后直接使用logger.Error, logger.Info操作默认的日志对象。
// 也可以用logger.New 创建一个自己的日志对象。
package logger

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"time"
)

//logging 是一个默认的日志对象，提供全局的Error, Info函数供使用，必须调用InitLogging
//函数进行初始化
var logging *Logger

func init() {
	logging = New("", true, false, DEBUG, 3, STDOUT)
}

var DEBUG = 0
var INFO = 3
var WARNING = 5
var ERROR = 7

var FILE = 0
var STDOUT = 3
var FILESTDOUT = 5

//InitLogging 初始化默认的日志对象，初始化后，就能使用Error，Info函数记录日志
func InitLogging(inputfilename string, level int, logType int) {
	logging = New(inputfilename, true, false,
		level, 3, logType)
}

// CloseDefault 把默认的日志关闭
func CloseDefault() {
	logging.Close()
}

//Error 默认日志对象方法，记录一条错误日志，需要先初始化
func Errorf(format string, v ...interface{}) {
	logging.Errorf(format, v...)
}

//Errorln 默认日志对象方法，记录一条消息日志，需要先初始化
func Errorln(args ...interface{}) {
	logging.Errorln(args...)
}

//Warningf 默认日志对象方法，记录一条错误日志，需要先初始化
func Warningf(format string, v ...interface{}) {
	logging.Warningf(format, v...)
}

//Warningln 默认日志对象方法，记录一条消息日志，需要先初始化
func Warningln(args ...interface{}) {
	logging.Warningln(args...)
}

//Infof 默认日志对象方法，记录一条消息日志，需要先初始化
func Infof(format string, v ...interface{}) {
	logging.Infof(format, v...)
}

//Infoln 默认日志对象方法，记录一条消息日志，需要先初始化
func Infoln(args ...interface{}) {
	logging.Infoln(args...)
}

//Debugf 默认日志对象方法，记录一条消息日志，需要先初始化
func Debugf(format string, v ...interface{}) {
	logging.Debugf(format, v...)
}

//Debugln 默认日志对象方法，记录一条调试日志，需要先初始化
func Debugln(args ...interface{}) {
	logging.Debugln(args...)
}

type Logger struct {
	level         int // debug 0 info 3 err 5
	innerLogger   *log.Logger
	curFile       *os.File
	todaydate     string
	filename      string
	runtimeCaller int
	isLogFilePath bool
	isLogFunc     bool
	msgQueue      chan string // 所有的日志先到这来
	closedCtx     context.Context
	closedCancel  context.CancelFunc
}

//New 创建一个自己的日志对象。
// filename:在logs文件夹下创建的文件名
// logFilePath: 日志中记录文件路径
// logFunc: 日志中记录调用函数
// level: 打印等级。DEBUG, INFO, ERROR
// callerDepth: 文件路径深度，设定适当的值，否则文件路径不正确
func New(filename string, logFilePath bool,
	logFunc bool, level int, callerDepth int, logType int) *Logger {

	// result := newLogger(logFile, flag)
	result := new(Logger)
	result.msgQueue = make(chan string, 1000)
	result.closedCtx, result.closedCancel = context.WithCancel(context.Background())

	var multi io.Writer

	if logType == FILE || logType == FILESTDOUT {
		if filename == "" {
			panic("logger init failed, filepath is emtpy")
		}

		dir, _ := filepath.Abs(filepath.Dir(os.Args[0]))
		filepath := dir + "/logs/" + filename
		fmt.Println("logger filepath", filepath)
		logFile, err := os.OpenFile(filepath,
			os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0666)
		if err != nil {
			panic(err.Error())
		}
		result.curFile = logFile

		if logType == FILESTDOUT {
			fmt.Println("newLogger use MultiWriter, file and stdout")
			multi = io.MultiWriter(logFile, os.Stdout)
		} else {
			fmt.Println("newLogger use file")
			multi = logFile
		}
	} else if logType == STDOUT {
		result.curFile = nil
		fmt.Println("newLogger use stdout")
		multi = os.Stdout
	}

	result.innerLogger = log.New(multi, "", 0)
	result.filename = filename
	result.runtimeCaller = callerDepth
	result.isLogFilePath = logFilePath
	result.isLogFunc = logFunc
	result.level = level
	result.todaydate = time.Now().Format("2006-01-02")

	// 启动日志切换
	go result.logworker()
	return result
}

// Close 关闭这一个日志对象
func (logobj *Logger) Close() error {
	logobj.closedCancel()
	close(logobj.msgQueue)
	return nil
}

func (logobj *Logger) getFormat(prefix, format string) string {
	var buf bytes.Buffer

	// 增加时间
	buf.WriteString(time.Now().Format("2006-01-02 15:04:05 "))
	buf.WriteString(prefix)

	// 增加文件和行号
	funcName, file, line, ok := runtime.Caller(logobj.runtimeCaller)
	if ok {
		if logobj.isLogFilePath {
			buf.WriteString(filepath.Base(file))
			buf.WriteString(":")
			buf.WriteString(strconv.Itoa(line))
			buf.WriteString(" ")
		}
		if logobj.isLogFunc {
			buf.WriteString(runtime.FuncForPC(funcName).Name())
			buf.WriteString(" ")
		}
		buf.WriteString(format)
		format = buf.String()
	}
	return format
}

//Errorf 记录一条错误日志
func (logobj *Logger) Errorf(format string, v ...interface{}) {
	if logobj.level > ERROR {
		return
	}

	format = logobj.getFormat("ERROR ", format)
	select {
	case <-logobj.closedCtx.Done():
	default:
		logobj.msgQueue <- fmt.Sprintf(format, v...)
	}
}

//Errorln 打印一行错误日志
func (logobj *Logger) Errorln(args ...interface{}) {
	if logobj.level > ERROR {
		return
	}

	prefix := logobj.getFormat("ERROR ", "")
	select {
	case <-logobj.closedCtx.Done():
	default:
		logobj.msgQueue <- fmt.Sprint(append([]interface{}{prefix}, args...)...)
	}
}

//Warningf 记录一条错误日志
func (logobj *Logger) Warningf(format string, v ...interface{}) {
	if logobj.level > WARNING {
		return
	}

	format = logobj.getFormat("WARNING ", format)
	select {
	case <-logobj.closedCtx.Done():
	default:
		logobj.msgQueue <- fmt.Sprintf(format, v...)
	}
}

//Warningln 打印一行错误日志
func (logobj *Logger) Warningln(args ...interface{}) {
	if logobj.level > WARNING {
		return
	}

	prefix := logobj.getFormat("WARNING ", "")
	select {
	case <-logobj.closedCtx.Done():
	default:
		logobj.msgQueue <- fmt.Sprint(append([]interface{}{prefix}, args...)...)
	}
}

//Infof 记录一条消息日志
func (logobj *Logger) Infof(format string, v ...interface{}) {
	if logobj.level > INFO {
		return
	}

	format = logobj.getFormat("INFO ", format)
	select {
	case <-logobj.closedCtx.Done():
	default:
		logobj.msgQueue <- fmt.Sprintf(format, v...)
	}
}

//Infoln 打印一行消息日志
func (logobj *Logger) Infoln(args ...interface{}) {
	if logobj.level > INFO {
		return
	}

	prefix := logobj.getFormat("INFO ", "")
	select {
	case <-logobj.closedCtx.Done():
	default:
		logobj.msgQueue <- fmt.Sprint(append([]interface{}{prefix}, args...)...)
	}
}

//Debugf 记录一条消息日志
func (logobj *Logger) Debugf(format string, v ...interface{}) {
	if logobj.level > DEBUG {
		return
	}

	format = logobj.getFormat("DEBUG ", format)
	select {
	case <-logobj.closedCtx.Done():
	default:
		logobj.msgQueue <- fmt.Sprintf(format, v...)
	}
}

//Debugln 打印一行调试日志
func (logobj *Logger) Debugln(args ...interface{}) {
	if logobj.level > DEBUG {
		return
	}

	prefix := logobj.getFormat("DEBUG ", "")
	select {
	case <-logobj.closedCtx.Done():
	default:
		logobj.msgQueue <- fmt.Sprint(append([]interface{}{prefix}, args...)...)
	}
}

func (logobj *Logger) logworker() {
	for msg := range logobj.msgQueue {
		logobj.innerLogger.Println(msg)
		//跨日改时间，后台启动
		nowDate := time.Now().Format("2006-01-02")
		if nowDate != logobj.todaydate {
			logobj.Debugf("doRotate run %v %v", nowDate, logging.todaydate)
			logobj.doRotate()
		}
	}
}

func (logobj *Logger) doRotate() {
	// 日志按天切换文件，日志对象记录了程序启动时的时间，当当前时间和程序启动的时间不一致
	// 则会启动到这个函数来改变文件
	// 首先关闭文件句柄，把当前日志改名为昨天，再创建新的文件句柄，将这个文件句柄赋值给log对象
	// 最后尝试删除5天前的日志
	fmt.Println("doRotate run")

	defer func() {
		rec := recover()
		if rec != nil {
			fmt.Printf("doRotate %v", rec)
		}
	}()

	if logobj.curFile == nil {
		fmt.Println("doRotate curfile nil, return")
		return
	}

	dir, _ := filepath.Abs(filepath.Dir(os.Args[0]))
	prefile := logobj.curFile

	_, err := prefile.Stat()
	if err == nil {
		filePath := dir + "/logs/" + logobj.filename

		err := prefile.Close()
		fmt.Printf("doRotate close err %v", err)
		nowTime := time.Now()
		time1dAgo := nowTime.Add(-1 * time.Hour * 24)
		err = os.Rename(filePath, filePath+"."+time1dAgo.Format("2006-01-02"))
		fmt.Printf("doRotate rename err %v", err)
	}

	if logobj.filename != "" {
		nextfile, err := os.OpenFile(dir+"/logs/"+logobj.filename,
			os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0666)
		if err != nil {
			fmt.Println(err.Error())
		}
		logobj.curFile = nextfile

		fmt.Println("newLogger use MultiWriter")
		multi := io.MultiWriter(nextfile, os.Stdout)
		logobj.innerLogger.SetOutput(multi)
	}

	fmt.Println("doRotate ending")

	// 更新标记，这个标记决定是否会启动文件切换
	nowDate := time.Now().Format("2006-01-02")
	logobj.todaydate = nowDate
	logobj.deleteHistory()
}

func (logobj *Logger) deleteHistory() {
	// 尝试删除5天前的日志
	fmt.Println("deleteHistory run")
	nowTime := time.Now()
	time5dAgo := nowTime.Add(-1 * time.Hour * 24 * 5)

	dir, _ := filepath.Abs(filepath.Dir(os.Args[0]))
	filePath := dir + "/logs/" + logobj.filename + "." + time5dAgo.Format("2006-01-02")

	_, err := os.Stat(filePath)
	if err == nil {
		os.Remove(filePath)
	}
}
