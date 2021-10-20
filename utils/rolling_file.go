package utils

import (
	"os"
	"path"
	"strings"
	"time"
)

const interval = 15 // roll every 15 minutes

type RollingFile struct {
	dir      string
	filename string
	file     *os.File
	ch       chan string
	ticker   *time.Ticker
	signals  chan os.Signal
}

func NewRollingFile(dir string, filename string) *RollingFile {
	file_path := path.Join(dir, filename)
	file, err := os.OpenFile(file_path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		panic(err)
	}

	ch := make(chan string)
	ticker := time.NewTicker(time.Minute) // check every minute
	signals := make(chan os.Signal, 1)    // roll if SIGHUP received
	rf := &RollingFile{dir, filename, file, ch, ticker, signals}

	go func() {
		for {
			select {
			case text := <-ch:
				if _, err = rf.file.WriteString(text); err != nil {
					panic(err)
				}
			case <-ticker.C:
				now := time.Now().Unix()
				if now/60%interval == 0 {
					rf.roll()
				}
			case <-signals:
				rf.roll()
			}
		}
	}()
	return rf
}

func (rf *RollingFile) roll() {
	rf.file.Sync()
	rf.file.Close()
	minute := time.Now().Format(time.RFC3339)[:16]
	minute = strings.ReplaceAll(minute, "T", "-")
	minute = strings.ReplaceAll(minute, ":", "-")
	file_path := path.Join(rf.dir, rf.filename)
	new_file_path := path.Join(rf.dir, rf.filename+"."+minute+".json")
	err := os.Rename(file_path, new_file_path)
	if err != nil {
		panic(err)
	}
	file, err := os.OpenFile(file_path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		panic(err)
	}
	rf.file = file
}

func (rf *RollingFile) Write(line string) {
	rf.ch <- line
}

func (rf *RollingFile) Close() {
	close(rf.ch)
	close(rf.signals)
	rf.ticker.Stop()
	rf.file.Sync()
	rf.file.Close()
}
