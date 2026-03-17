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
	Records map[string][]*payload.Record
	NextNo  uint64
	File    string
	stopCh  chan struct{}
}

func NewStore(file string) *Store {
	return &Store{
		Records: make(map[string][]*payload.Record),
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

	records := make(map[string][]*payload.Record)
	var counter uint64

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
		values := fields[2:]

		fqdn := dns.Fqdn(name)

		valuePtrs := make([]*string, len(values))
		for i, v := range values {
			val := v
			valuePtrs[i] = &val
		}

		no := counter
		counter++
		nameCopy := name
		typCopy := typ

		records[fqdn] = append(records[fqdn], &payload.Record{
			No:     &no,
			Name:   &nameCopy,
			Type:   &typCopy,
			Values: valuePtrs,
		})
	}

	r.Mu.Lock()
	r.Records = records
	r.NextNo = counter
	r.Mu.Unlock()
}

func (r *Store) Save() {
	r.Mu.RLock()
	defer r.Mu.RUnlock()

	f, err := os.Create(r.File)
	if err != nil {
		return
	}
	defer f.Close()

	w := bufio.NewWriter(f)
	for _, records := range r.Records {
		for _, r := range records {
			line := *r.Name + " " + *r.Type
			for _, v := range r.Values {
				line += " " + *v
			}
			_, _ = w.WriteString(line + "\n")
		}
	}
	_ = w.Flush()
}

func (r *Store) Tick() {
	ticker := time.NewTicker(30 * time.Second)
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
