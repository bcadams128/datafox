package utils

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"
)

func RunUploader(ctx context.Context, dir string, om *ObjectManager) {
	done := make(map[string]bool)
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		uploadPending(ctx, dir, om, done)

		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func uploadPending(ctx context.Context, dir string, om *ObjectManager, done map[string]bool) {
	paths, err := filepath.Glob(filepath.Join(dir, "*.parquet"))
	if err != nil {
		log.Printf("uploader glob failed: %v", err)
		return
	}

	for _, path := range paths {
		if done[path] {
			continue
		}
		if err := uploadFile(ctx, path, om); err != nil {
			log.Printf("upload failed: %v", err)
			continue
		}
		done[path] = true
	}
}

func uploadFile(ctx context.Context, path string, om *ObjectManager) error {
	key := filepath.Base(path)

	localchecksum, err := fileChecksum(path)
	if err != nil {
		return fmt.Errorf("checksum %q: %w", path, err)
	}

	remotechecksum, found, err := om.RemoteChecksum(ctx, key)
	if err != nil {
		return err
	}
	if found && remotechecksum == localchecksum {
		return nil
	}
	if found {
		log.Printf("checksum mismatch for %s, re-uploading", key)
	}

	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open %q: %w", path, err)
	}
	defer f.Close()

	if err := om.Upload(ctx, key, f, path); err != nil {
		return err
	}
	log.Printf("uploaded %s", key)
	return nil
}
