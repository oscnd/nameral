package store

import (
	"bufio"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/miekg/dns"
	"go.scnd.dev/open/nameral/type/payload"
)

type Store struct {
	Mu      sync.RWMutex
	Records map[string]*payload.Record
	File    string
	stopCh  chan struct{}
}

func NewStore(file string) *Store {
	return &Store{
		Records: make(map[string]*payload.Record),
		File:    file,
		stopCh:  make(chan struct{}),
	}
}

func (r *Store) Load() {
	f, err := os.Open(r.File)
	if err != nil {
		return
	}
	defer f.Close()

	records := make(map[string]*payload.Record)

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}

		name := fields[0]
		typ := strings.ToUpper(fields[1])
		value := strings.Join(fields[2:], " ")

		fqdn := dns.Fqdn(name)
		h := HashRecord(fqdn, typ, value)

		records[h] = &payload.Record{
			Hash:  &h,
			Name:  &fqdn,
			Type:  &typ,
			Value: &value,
		}
	}

	r.Mu.Lock()
	r.Records = records
	r.Mu.Unlock()
}

func (r *Store) AddRecord(name, typ, value string) (string, error) {
	r.Mu.Lock()
	defer r.Mu.Unlock()

	fqdn := dns.Fqdn(name)
	h := HashRecord(fqdn, typ, value)

	r.Records[h] = &payload.Record{
		Hash:  &h,
		Name:  &fqdn,
		Type:  &typ,
		Value: &value,
	}

	newLine := fqdn + " " + typ + " " + value
	if err := r.appendLine(newLine); err != nil {
		return h, err
	}

	return h, nil
}

func (r *Store) DeleteRecordByHash(hash string) error {
	r.Mu.Lock()
	defer r.Mu.Unlock()

	rec, ok := r.Records[hash]
	if !ok {
		return nil
	}

	name := *rec.Name
	typ := *rec.Type
	value := *rec.Value

	lines, err := r.readAllLines()
	if err != nil {
		return err
	}

	newLines := make([]string, 0, len(lines))
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) < 3 {
			newLines = append(newLines, line)
			continue
		}
		lineName := fields[0]
		lineTyp := strings.ToUpper(fields[1])
		lineVal := strings.Join(fields[2:], " ")

		if lineName == name && lineTyp == typ && lineVal == value {
			continue
		}
		newLines = append(newLines, line)
	}

	if err := r.writeAllLines(newLines); err != nil {
		return err
	}

	delete(r.Records, hash)
	return nil
}

func (r *Store) GetRecordByHash(hash string) *payload.Record {
	r.Mu.RLock()
	defer r.Mu.RUnlock()

	return r.Records[hash]
}

func (r *Store) UpdateRecordByHash(hash, typ, value string) bool {
	r.Mu.Lock()
	defer r.Mu.Unlock()

	rec, ok := r.Records[hash]
	if !ok {
		return false
	}

	name := *rec.Name
	oldValue := *rec.Value

	rec.Type = &typ
	rec.Value = &value

	lines, err := r.readAllLines()
	if err != nil {
		return false
	}

	newLines := make([]string, 0, len(lines))
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) < 3 {
			newLines = append(newLines, line)
			continue
		}
		lineName := fields[0]
		lineTyp := strings.ToUpper(fields[1])
		lineVal := strings.Join(fields[2:], " ")

		if lineName == name && lineTyp == typ && lineVal == oldValue {
			newLines = append(newLines, name+" "+typ+" "+value)
		} else {
			newLines = append(newLines, line)
		}
	}

	if err := r.writeAllLines(newLines); err != nil {
		return false
	}

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

func (r *Store) appendLine(line string) error {
	f, err := os.OpenFile(r.File, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.WriteString(line + "\n")
	return err
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
