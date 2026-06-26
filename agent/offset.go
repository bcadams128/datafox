package main

import (
	"log"
	"os"

	"github.com/vmihailenco/msgpack/v5"
)

type OffsetState struct {
	Path   string `msgpack:"Path"`
	Inode  uint64 `msgpack:"inode"`
	Offset int64  `msgpack:"Offset"`
}

type Offset struct {
	Version int
	Files   map[string]OffsetState
}

func SaveOffsets(path string, db *Offset) error {
	tmp := path + ".tmp"
	//TODO make this a debug log
	log.Print("Saving offset to ", tmp)

	b, err := msgpack.Marshal(db)
	if err != nil {
		return err
	}

	if err := os.WriteFile(tmp, b, 0644); err != nil {
		return err
	}

	return os.Rename(tmp, path)
}

func LoadOffsets(path string) (*Offset, error) {
	log.Print("Checking for existing offsets in:", path)
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			log.Print("Existing offset not found, creating a new offset state")
			return &Offset{Files: make(map[string]OffsetState)}, nil
		}
		return nil, err
	}

	var db Offset
	if err := msgpack.Unmarshal(b, &db); err != nil {
		return nil, err
	}

	if db.Files == nil {
		db.Files = make(map[string]OffsetState)
	}

	log.Printf("Database version: %d", db.Version)

	if len(db.Files) > 0 {
		log.Printf("Loaded offsets for %d files:", len(db.Files))
		for filePath, state := range db.Files {
			log.Printf("  - %s: offset=%d, inode=%d", filePath, state.Offset, state.Inode)
		}
	} else {
		log.Print("No existing offsets found in backup file")
	}
	return &db, nil
}

func TailersToOffsets(tailers []*Tailer) *Offset {
	o := &Offset{Files: make(map[string]OffsetState)}
	for _, t := range tailers {
		o.Version = 1
		o.Files[t.path] = t.State()
	}
	return o
}
