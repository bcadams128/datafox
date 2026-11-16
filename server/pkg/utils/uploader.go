package utils

import (
	"context"
	"datafox/server/pkg/cache"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"
)

func RunUploader(ctx context.Context, dir string, cache *cache.Cache, om *ObjectManager) {
	done := make(map[string]bool)
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		uploadPending(ctx, dir, cache, om, done)

		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func uploadPending(ctx context.Context, dir string, cache *cache.Cache, om *ObjectManager, done map[string]bool) {
	paths, err := filepath.Glob(filepath.Join(dir, "*.parquet"))
	if err != nil {
		log.Printf("uploader glob failed: %v", err)
		return
	}

	for _, path := range paths {
		if done[path] {
			continue
		}
		if err := uploadFile(ctx, path, cache, om); err != nil {
			log.Printf("upload failed: %v", err)
			continue
		}
		done[path] = true
	}
}

func uploadFile(ctx context.Context, path string, cache *cache.Cache, om *ObjectManager) error {
	base := filepath.Base(path)
	host := "host01"
	t, err := time.Parse("logs-20060102-150405.000.parquet", base)
	year, month, day := t.Date()
	key := fmt.Sprintf("%s/%d/%d/%d/%d/%d/%s", host, year, month, day, t.Hour(), t.Minute(), base)

	localchecksum, err := fileChecksum(path)
	if err != nil {
		return fmt.Errorf("checksum %q: %w", path, err)
	}

	remotechecksum, found, err := om.RemoteChecksum(ctx, key)
	if err != nil {
		return err
	}

	if found && remotechecksum == localchecksum {
		log.Printf("already uploaded")
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
