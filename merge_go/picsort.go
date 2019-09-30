package main

import (
	"crypto/sha256"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/xor-gate/goexif2/exif"
)

// ErrCopy is just a common error
var ErrCopy = errors.New("Error copying file, repeat later")

// Files data
type filesData struct {
	name string
	date time.Time
}

func fHash(picName string) string {
	f, err := os.Open(picName)
	if err != nil {
		log.Fatal(err)
		return "0"
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		log.Fatal(err)
		return "0"
	}

	return fmt.Sprintf("%x", h.Sum(nil))
}

func move(src, dst string) (int64, error) {
	sourceFileStat, err := os.Stat(src)
	if err != nil {
		return 0, err
	}

	if !sourceFileStat.Mode().IsRegular() {
		return 0, fmt.Errorf("%s is not a regular file", src)
	}

	source, err := os.Open(src)
	if err != nil {
		return 0, err
	}
	defer source.Close()

	remotePath := filepath.Dir(dst)
	if _, err := os.Stat(remotePath); os.IsNotExist(err) {
		os.MkdirAll(remotePath, 0777)
	}

	if _, err := os.Stat(dst); os.IsExist(err) {
		if fHash(src) == fHash(dst) {
			return 0, nil
		}

		dst = filepath.Join(remotePath, fmt.Sprintf("%v_01.%v", strings.Split(src, ".")[0], filepath.Ext(src)))
	}

	destination, err := os.Create(dst)
	if err != nil {
		return 0, err
	}
	defer destination.Close()
	nBytes, err := io.Copy(destination, source)

	// validate
	if fHash(src) != fHash(dst) {
		os.Remove(dst)
		err = ErrCopy
	}

	if err == nil {
		fmt.Printf("Moved %s to %s\n", src, dst)
	}

	return nBytes, err
}

func buildExtMap(exts []string) map[string]bool {
	var m map[string]bool = make(map[string]bool)
	for _, ext := range exts {
		m[ext] = true
	}

	return m
}

func validExtention(path string) bool {
	extMap := buildExtMap([]string{".jpg", ".jpeg", ".JPG", ".gif"})
	if _, ok := extMap[filepath.Ext(path)]; ok {
		return true
	}

	return false
}

func findPictures(path string, files chan filesData) {
	defer close(files)
	err := filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}
		if validExtention(path) {
			date, err := getDate(path)
			if err != nil {
				fmt.Println(err)
			} else {
				files <- filesData{name: path, date: date}
			}
		}
		return nil
	})
	if err != nil {
		panic(err)
	}
}

func getDate(picName string) (time.Time, error) {
	// try figure out GIFs.
	r, _ := regexp.Compile(`Burst_Cover_GIF_Action_\d+\.gif`)
	if r.MatchString(picName) {
		r, _ = regexp.Compile(`_\d+`)
		possibleDate := strings.Replace(r.FindString(picName), "_", "", -1)
		dateTime, err := parsePossibleDate(possibleDate)
		return dateTime, err
	}

	/// get EXIF data
	f, err := os.Open(picName)
	if err != nil {
		//log.Fatal(err)
		return time.Date(1900, time.Month(1), 1, 0, 0, 0, 0, time.UTC), err
	}
	// Close the file
	defer f.Close()

	var dateTime time.Time
	tags, err := exif.Decode(f)
	if err != nil {
		// try to parse the file name
		r, _ := regexp.Compile(`IMG-\d+-.*\.jpg`)
		if r.MatchString(picName) {
			r, _ = regexp.Compile(`-\d+-`)
			possibleDate := strings.Replace(r.FindString(picName), "-", "", -1)
			dateTime, err = parsePossibleDate(possibleDate)
			if err != nil {
				fmt.Printf("File: %v doesn't have the DateTime field and can't parse the date!\n", picName)
			}
			return dateTime, err
		}
	} else {
		dateTime, err = tags.DateTime()
		if err != nil {
			return time.Date(1900, time.Month(1), 1, 0, 0, 0, 0, time.UTC), nil
		}
	}

	return dateTime, err
}

func parsePossibleDate(possibleDate string) (time.Time, error) {
	// split first
	splitDate := strings.Split(possibleDate, "")

	// first 4 digits are a year, then a month, then a day
	year, _ := strconv.Atoi(strings.Join(splitDate[0:4], ""))
	month, _ := strconv.Atoi(strings.Join(splitDate[4:6], ""))
	day, _ := strconv.Atoi(strings.Join(splitDate[6:8], ""))

	// try to parse the date
	return time.Date(year, time.Month(month), day, 4, 0, 0, 0, time.UTC), nil
}

func moveFiles(i int, collect string, files chan filesData, wg *sync.WaitGroup) {
	defer wg.Done()
	for fd := range files {
		dateTime := fd.date
		filePath := fmt.Sprintf("%v/%v/%02d/%02d", collect, dateTime.Year(), dateTime.Month(), dateTime.Day())
		picName := filepath.Join(filePath, filepath.Base(fd.name))

		// move
		if _, err := move(fd.name, picName); err == ErrCopy {
			// copy failed
			fmt.Printf("Error copying %v, try again later\n", picName)
			files <- fd
		}
	}
}

func main() {

	target := flag.String("target", "", "Target folder to copy files to")
	flag.Parse()

	files := make(chan filesData)

	// Make dir for the files, call picsToCopy
	collectPath := filepath.Join("/tmp", "picsToCopy")
	os.MkdirAll(collectPath, 0777)

	go findPictures(".", files)

	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go moveFiles(i, collectPath, files, &wg)
	}
	wg.Wait()

	// Run rsync
	fmt.Printf("Now run 'rsync -n --size-only --progress -ruvzh --no-perms --chmod=ugo=rwX %v/ %v' to see the list of files to transfer, then remove '-n' and run again\n", collectPath, *target)
}
