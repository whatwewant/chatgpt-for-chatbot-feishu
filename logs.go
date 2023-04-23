package main

import (
	"os"

	"github.com/go-zoox/core-utils/fmt"
	"github.com/go-zoox/fs"
	"github.com/go-zoox/logger"
	"github.com/go-zoox/logger/components/transport"
	"github.com/go-zoox/logger/transport/file"
)

type Logs struct {
	Dir   string
	Level string

	//
	accessLogFile *os.File
	errorLogFile  *os.File
	debugLogFile  *os.File
}

func (l *Logs) Setup() (err error) {
	if ok := fs.IsExist(l.Dir); !ok {
		if err := fs.Mkdirp(l.Dir); err != nil {
			return fmt.Errorf("failed to create log directory for : %v", err)
		}
	}

	accessLog := fs.JoinPath(l.Dir, "access.log")
	if l.accessLogFile, err = fs.Open(accessLog); err != nil {
		return fmt.Errorf("failed to open access log(%s): %v", accessLog, err)
	}
	errorLog := fs.JoinPath(l.Dir, "error.log")
	if l.errorLogFile, err = fs.Open(errorLog); err != nil {
		return fmt.Errorf("failed to open error log(%s): %v", accessLog, err)
	}
	debugLog := fs.JoinPath(l.Dir, "debug.log")
	if l.debugLogFile, err = fs.Open(debugLog); err != nil {
		return fmt.Errorf("failed to open debug log(%s): %v", accessLog, err)
	}

	logger.AppendTransports(map[string]transport.Transport{
		"access": file.New(&file.Config{
			Level: "info",
			File:  l.accessLogFile,
			Exact: true,
		}),
		"error": file.New(&file.Config{
			Level: "error",
			File:  l.errorLogFile,
			Exact: true,
		}),
		"debug": file.New(&file.Config{
			Level: "debug",
			File:  l.debugLogFile,
			Exact: true,
		}),
	})

	logger.SetLevel(l.Level)

	return nil
}
