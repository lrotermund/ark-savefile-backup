package main

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"bufio"

	"github.com/fsnotify/fsnotify"
)

var files = [...]string{"TheIsland.ark", "LocalPlayer.arkprofile", "PlayerLocalData.arkprofile"}
var path = ""

func main() {
	fmt.Println("Ark save file backup")
	fmt.Println("-----------------------")
	path = getPathToSaveFile()
	watcher, err := fsnotify.NewWatcher()

	if err != nil {
		log.Fatal(err)
	}

	defer watcher.Close()

	done := make(chan bool)

	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				log.Println("event:", event)
				if event.Op&fsnotify.Write == fsnotify.Write {
					onModifiedFile(event)
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.Println("error:", err)
			}
		}
	}()

	fmt.Println("Start watching:")
	for _, file := range files {
		fmt.Printf("- %s%s", file, getCrlf())
	}

	if err := watcher.Add(path); err != nil {
		log.Fatal(err)
	}

	<-done
}

func getPathToSaveFile() string {
	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Println("Please insert the folder path to your Ark save files.")
		path := filepath.FromSlash(getPath(reader))

		if validatePath(path) {
			return path
		}
	}
}

func getPath(reader *bufio.Reader) string {
	fmt.Print("-> ")
	pathWithCrlf, _ := reader.ReadString('\n')
	crlf := getCrlf()
	return strings.Replace(pathWithCrlf, crlf, "", -1)
}

func getCrlf() string {
	if runtime.GOOS == "windows" {
		return "\r\n"
	}

	return "\n"
}

func validatePath(path string) bool {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return false
	}

	return true
}

func onModifiedFile(event fsnotify.Event) {
	modifiedFile := event.Name
	log.Println("modified file:", modifiedFile)

	if !shouldHandleFile(modifiedFile) {
		log.Println("the file is not included in the watch list and is ignored:", modifiedFile)
		return
	}

	log.Println("file is backed up:", modifiedFile)
	createBackup(modifiedFile)
}

func shouldHandleFile(file string) bool {
	pathSeperator := string(os.PathSeparator)

	baseFilePath := path

	if !strings.HasSuffix(baseFilePath, pathSeperator) {
		baseFilePath += pathSeperator
	}

	for _, watchedFile := range files {
		baseFilePath = fmt.Sprintf("%s%s", baseFilePath, watchedFile)

		if file == baseFilePath {
			return true
		}
	}
	return false
}

func createBackup(file string) {
	basePath := createFolderIfItNotExist(path, "ark_savefile_backups")
	currentTime := time.Now()
	backupPath := createFolderIfItNotExist(basePath, currentTime.Format("2006-01-02_15:04"))

	for _, file := range files {
		createFileBackup(file, backupPath)
	}
}

func createFolderIfItNotExist(basePath string, name string) string {
	pathSeperator := string(os.PathSeparator)

	if !strings.HasSuffix(basePath, pathSeperator) {
		basePath += pathSeperator
	}

	newFolderPath := fmt.Sprintf("%s%s", basePath, name)

	if err := os.MkdirAll(newFolderPath, os.ModePerm); err != nil {
		log.Fatal(err)
	}

	return newFolderPath
}

func createFileBackup(file string, folderPath string) {
	pathSeperator := string(os.PathSeparator)
	baseFilePath := path

	if !strings.HasSuffix(baseFilePath, pathSeperator) {
		baseFilePath += pathSeperator
	}

	baseFilePath = fmt.Sprintf("%s%s", baseFilePath, file)

	if _, err := os.Stat(baseFilePath); os.IsNotExist(err) {
		log.Println("Ark file could not be loaded - The file is skipped:", file)
		return
	}

	if !strings.HasSuffix(folderPath, pathSeperator) {
		folderPath += pathSeperator
	}

	filePath := fmt.Sprintf("%s%s", folderPath, file)
	backupFile, err := os.Create(filePath)

	if err != nil {
		log.Fatal(err)
	}

	defer backupFile.Close()

	baseFile, err := ioutil.ReadFile(baseFilePath)

	if err != nil {
		log.Fatal(err)
	}

	if _, err := io.Copy(backupFile, bytes.NewReader(baseFile)); err != nil {
		log.Fatal(err)
	}
}
