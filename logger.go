package main

import (
	"fmt"
	"log"
	"os"
	"path"
	"time"
)

/**
 * Sets up the global logger for the app
 *
 * @author Ben Reichelt <ben.reichelt@gmail.com>
 *
 * @return  Logger
**/

func createLogger() *log.Logger {

	t := time.Now()
	format := "20060102"

	filePart := t.Format(format)

	logDir := ensureUserHomeDir()
	logDir = path.Join(logDir, "logs")
	err := os.MkdirAll(logDir, perms)
	if err != nil {
		panic(err)
	}

	filePath := path.Join(logDir, "fsync-"+filePart+".log")
	var fi *os.File
	if !pathExists(filePath) {
		fi, _ = os.Create(filePath)
	} else {
		fi, _ = os.OpenFile(filePath, os.O_RDWR|os.O_APPEND, 0755)
	}

	l := log.New(fi, "", log.Ldate|log.Ltime|log.Lmicroseconds|log.Lshortfile)
	return l
}

/**
 * Logs a message to the log file and optionally echos the message to stdout
 *
 * @author Ben Reichelt <ben.reichelt@gmail.com>
 *
 * @param   string   The message to log
 * @param   bool     Determines if we should echo the message as well
 * @return  void
**/

func logMessage(message string, echo bool) {

	Flogger.Println(message)
	if echo {
		fmt.Println(message)
	}
}
