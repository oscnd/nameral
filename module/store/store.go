package store

import (
	"bufio"
	"os"
	"strings"
	"sync"
	"time"

	"go.scnd.dev/open/nameral/type/payload"
)

type Store struct {
	Mu        sync.RWMutex
	Records   map[uint64]*payload.Record
	File      string
	stopCh    chan struct{}
	LineCount uint64
}

func NewStore(file string) *Store {
	return &Store{
		Records:   make(map[uint64]*payload.Record),
		File:      file,
		stopCh:    make(chan struct{}),
		LineCount: 0,
	}
}

func (r *Store) Load() {
	f, err := os.Open(r.File)
	if err != nil {
		return
	}
	defer f.Close()

	records := make(map[uint64]*payload.Record)
	var lineNo uint64 = 1

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// * check if this line is a comment, blank, or a record
		if line == "" || strings.HasPrefix(line, "#") {
			// * just count the line, don't create a record
			lineNo++
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 3 {
			// * not a valid record line, but still counts as a line
			lineNo++
			continue
		}

		name := fields[0]
		typ := strings.ToUpper(fields[1])
		values := fields[2:]

		valuePtrs := make([]*string, len(values))
		for i, v := range values {
			val := v
			valuePtrs[i] = &val
		}

		no := lineNo
		nameCopy := name
		typCopy := typ

		records[no] = &payload.Record{
			No:     &no,
			Name:   &nameCopy,
			Type:   &typCopy,
			Values: valuePtrs,
		}
		lineNo++
	}

	r.Mu.Lock()
	r.Records = records
	r.LineCount = lineNo - 1
	r.Mu.Unlock()
}

func (r *Store) WriteLine(lineNo uint64, name, typ string, values []string) error {
	r.Mu.Lock()
	defer r.Mu.Unlock()

	// * read all lines
	lines, err := r.readAllLines()
	if err != nil {
		return err
	}

	// * build the new line
	newLine := name + " " + typ
	for _, v := range values {
		newLine += " " + v
	}

	// * update or append the line
	if lineNo > 0 && lineNo <= uint64(len(lines)) {
		lines[lineNo-1] = newLine
	} else if lineNo == uint64(len(lines))+1 {
		lines = append(lines, newLine)
	} else {
		return nil
	}

	return r.writeAllLines(lines)
}

func (r *Store) AddRecord(name, typ string, values []string) (uint64, error) {
	r.Mu.Lock()
	defer r.Mu.Unlock()

	// * read current file to get actual line count
	lines, _ := r.readAllLines()
	lineNo := uint64(len(lines)) + 1

	nameCopy := name
	typCopy := typ
	valuePtrs := make([]*string, len(values))
	for i, v := range values {
		val := v
		valuePtrs[i] = &val
	}

	r.Records[lineNo] = &payload.Record{
		No:     &lineNo,
		Name:   &nameCopy,
		Type:   &typCopy,
		Values: valuePtrs,
	}
	r.LineCount = lineNo

	// * append the new line to file
	newLine := name + " " + typ
	for _, v := range values {
		newLine += " " + v
	}

	f, err := os.OpenFile(r.File, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return lineNo, err
	}
	defer f.Close()
	_, err = f.WriteString(newLine + "\n")
	return lineNo, err
}

// DeleteRecordByNo deletes a record by its file line number and reorders remaining records
func (r *Store) DeleteRecordByNo(no uint64) error {
	r.Mu.Lock()
	defer r.Mu.Unlock()

	// * read all lines from file
	lines, err := r.readAllLines()
	if err != nil {
		return err
	}

	// * remove the line at position no
	if no > 0 && no <= uint64(len(lines)) {
		lines = append(lines[:no-1], lines[no:]...)
	}

	// * write back to file
	if err := r.writeAllLines(lines); err != nil {
		return err
	}

	// * update in-memory records
	newRecords := make(map[uint64]*payload.Record)
	for oldNo, rec := range r.Records {
		if oldNo == no {
			continue
		}
		// * reorder: if record has higher line number, decrement it
		newNo := oldNo
		if oldNo > no {
			newNo--
		}
		rec.No = &newNo
		newRecords[newNo] = rec
	}
	r.Records = newRecords
	if r.LineCount > 0 {
		r.LineCount--
	}

	return nil
}

func (r *Store) GetRecordByNo(no uint64) *payload.Record {
	r.Mu.RLock()
	defer r.Mu.RUnlock()

	return r.Records[no]
}

func (r *Store) UpdateRecordByNo(no uint64, typ string, values []string) bool {
	r.Mu.Lock()
	defer r.Mu.Unlock()

	rec := r.Records[no]
	if rec == nil {
		return false
	}

	typCopy := typ
	valuePtrs := make([]*string, len(values))
	for j, v := range values {
		val := v
		valuePtrs[j] = &val
	}
	rec.Type = &typCopy
	rec.Values = valuePtrs
	return true
}

func (r *Store) readAllLines() ([]string, error) {
	f, err := os.Open(r.File)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var lines []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines, scanner.Err()
}

func (r *Store) writeAllLines(lines []string) error {
	f, err := os.Create(r.File)
	if err != nil {
		return err
	}
	defer f.Close()

	w := bufio.NewWriter(f)
	for _, line := range lines {
		_, _ = w.WriteString(line + "\n")
	}
	return w.Flush()
}

func (r *Store) Tick() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			r.Load()
		case <-r.stopCh:
			return
		}
	}
}

func (r *Store) Stop() {
	close(r.stopCh)
}
