package mylog

import (
	"flag"
	"fmt"
	"log"
	"os"
	"syscall"
	"time"
	"path"
	"strings"
)

var logFile = flag.String("log.file", "", "save log file path")

var lf *os.File

func redirectStderr(f *os.File) {
	err := syscall.Dup2(int(f.Fd()), int(os.Stderr.Fd()))
	if err != nil {
		log.Fatalf("Failed to redirect stderr to file: %v", err)
	}
}

func fileExist(filename string) bool {
	_, err := os.Stat(filename)
	return err == nil || os.IsExist(err)
}

func InitLog() {
	logfile := *logFile
	var err error
	log.SetFlags(log.Lshortfile | log.LstdFlags)
	if logfile != "" {
		if fileExist(logfile) {	
			fmt.Println("log file already exist:", logfile)		
			//t := time.Now().Format(layout)
			t := time.Now()
			year, month, day := t.Date()
			filename :=  path.Base(logfile)
			fileSuffix := path.Ext(filename) //获取文件后缀
			filename_olny := strings.TrimSuffix(filename, fileSuffix)
			logfile = path.Dir(logfile) + "/" + filename_olny +
					fmt.Sprintf("%d-%d-%d_%d-%d-%d", year, month, day, t.Hour(), t.Minute(), t.Second()) +
					fileSuffix
			fmt.Printf("create new log file name: %s\n", logfile)
		}

		lf, err = os.OpenFile(logfile, os.O_CREATE|os.O_RDWR, 0644)
		if err != nil {
			log.Fatalln("open log file error: ", err)
		}
		//redirectStderr(lf)
		log.SetOutput(lf)
	}
}

func Close() {
	if lf != nil {
		lf.Close()
	}
}