package disk

import (
	"os"
	"testing"
	"time"
)

func TestWriteBatch(t *testing.T) {
	dir := t.TempDir()
	t1 := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	t2 := t1.Add(5 * time.Minute)
	currentTime := t1

	writer := &DiskWriter{
		Dir: dir,
		Now: func() time.Time {
			return currentTime
		},
	}

	batch := LogBatch{
		Lines:    []string{"line one", "line two"},
		Metadata: map[string]string{"hostname": "myhost", "source": "var/log/app.log"},
	}

	t.Run("Disk Write Happy Path", func(t *testing.T) {
		if err := writer.WriteBatch(batch); err != nil {
			t.Fatalf("Write Batch errored: %v", err)
		}
	})

	t.Run("Rotation and file close", func(t *testing.T) {
		err := writer.WriteBatch(batch)
		if err != nil {
			t.Fatal(err)
		}

		firstFile := writer.current
		currentTime = t2

		err = writer.WriteBatch(batch)
		if err != nil {
			t.Fatal(err)
		}

		secondFile := writer.current
		if firstFile == secondFile {
			t.Fatalf("Files expected to be different.  File rotation failed %v %v", firstFile, secondFile)
		}

		_, err = firstFile.Write([]byte("fail"))
		if err == nil {
			t.Fatal("Expected file to be closed")
		}

	})

	t.Run("Diskworker Test", func(t *testing.T) {
		dir := t.TempDir()
		writer := &DiskWriter{
			Dir: dir,
			Now: func() time.Time {
				return currentTime
			},
		}
		testChan := make(chan LogBatch, 3)

		testChan <- batch
		testChan <- batch
		close(testChan)
		DiskWorker(testChan, writer)

		files, _ := os.ReadDir(dir)
		if len(files) != 1 {
			t.Fatalf("expected file got %d", len(files))
		}

	})
}
