package log

import (
	"fmt"
	"io"
	"log"
	"sync"

	"github.com/fatih/color"
)

var (
	yellow = color.New(color.FgHiYellow).SprintFunc()
	red    = color.New(color.FgHiRed, color.Bold).SprintFunc()
	brue   = color.New(color.FgHiBlue).SprintFunc()
	faint  = color.New(color.Faint).SprintFunc()

	infoColor  = color.New(color.FgHiWhite).SprintFunc()
	errorColor = color.New(color.FgHiRed).SprintFunc()
	debugColor = color.New(color.FgHiWhite).SprintFunc()
)

type MeloLog struct {
	infoLog  *log.Logger
	errorLog *log.Logger
	debugLog *log.Logger

	mu *sync.Mutex

	Enabled bool
}

type InfoWriter struct {
	out io.Writer
}

func (dw InfoWriter) Write(p []byte) (n int, err error) {
	p = faintFilePath(p)
	str := append([]byte(yellow("┏ ")), p...)

	return dw.out.Write(str)
}

type ErrorWriter struct {
	out io.Writer
}

func (dw ErrorWriter) Write(p []byte) (n int, err error) {
	p = faintFilePath(p)
	str := append([]byte(red("┏")), p...)

	return dw.out.Write(str)
}

type DebugWriter struct {
	out io.Writer
}

func (dw DebugWriter) Write(p []byte) (n int, err error) {
	p = faintFilePath(p)
	str := append([]byte(brue("┏")), p...)

	return dw.out.Write(str)
}

func New(out io.Writer) MeloLog {
	InfoLog := log.New(InfoWriter{out}, "\n"+yellow("┗► INFO ")+": ", log.Llongfile|log.Lmsgprefix)
	ErrorLog := log.New(ErrorWriter{out}, "\n"+red("┗► ERROR")+": ", log.Llongfile|log.Lmsgprefix)
	DebugLog := log.New(DebugWriter{out}, "\n"+brue("┗► DEBUG")+": ", log.Llongfile|log.Lmsgprefix)

	return MeloLog{
		infoLog:  InfoLog,
		errorLog: ErrorLog,
		debugLog: DebugLog,
		Enabled:  true,
		mu:       &sync.Mutex{},
	}
}

func (l *MeloLog) Info(msg string) {
	if l.Enabled {
		l.infoLog.Output(2, infoColor(msg))
	}
}

func (l *MeloLog) Error(msg string) {
	if l.Enabled {
		l.errorLog.Output(2, errorColor(msg))
	}
}

func (l *MeloLog) Debug(msg string) {
	if l.Enabled {
		l.debugLog.Output(2, debugColor(msg))
	}
}

func (l *MeloLog) Infof(format string, args ...any) {
	if l.Enabled {
		l.infoLog.Output(2, fmt.Sprintf(infoColor(format), args...))
	}
}

func (l *MeloLog) Errorf(format string, args ...any) {
	if l.Enabled {
		l.errorLog.Output(2, fmt.Sprintf(errorColor(format), args...))
	}
}

func (l *MeloLog) Debugf(format string, args ...any) {
	if l.Enabled {
		l.debugLog.Output(2, fmt.Sprintf(debugColor(format), args...))
	}
}

func (l *MeloLog) SetEnabled(boolean bool) {
	l.mu.Lock()
	l.Enabled = boolean
	l.mu.Unlock()
}
