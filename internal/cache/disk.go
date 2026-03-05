package cache

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"time"
)

const defaultTTL = 4 * time.Hour

// DiskCacher implements github.Cacher by storing entries as files on disk.
// Each cache key is hashed to a SHA-256 hex filename. Staleness is determined
// by file mtime against a configurable TTL.
type DiskCacher struct {
	dir string
	ttl time.Duration
}

// NewDiskCacher creates a DiskCacher rooted at dir with the given TTL.
// If dir is empty, it defaults to os.UserCacheDir()/gh-inbox.
// If ttl is zero, it defaults to 4 hours.
// The directory is created if it does not exist.
func NewDiskCacher(dir string, ttl time.Duration) (*DiskCacher, error) {
	if dir == "" {
		base, err := os.UserCacheDir()
		if err != nil {
			return nil, err
		}
		dir = filepath.Join(base, "gh-inbox")
	}
	if ttl == 0 {
		ttl = defaultTTL
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	return &DiskCacher{dir: dir, ttl: ttl}, nil
}

// Get returns cached data for key. A missing or stale file is a cache miss.
func (d *DiskCacher) Get(key string) ([]byte, bool, error) {
	path := d.filepath(key)
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	if time.Since(info.ModTime()) > d.ttl {
		return nil, false, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, false, err
	}
	return data, true, nil
}

// Set writes data to the cache for key using an atomic rename.
func (d *DiskCacher) Set(key string, data []byte) error {
	path := d.filepath(key)
	tmp, err := os.CreateTemp(d.dir, "tmp-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return err
	}
	if err := os.Rename(tmpName, path); err != nil {
		os.Remove(tmpName)
		return err
	}
	return nil
}

// filepath returns the on-disk path for a cache key.
func (d *DiskCacher) filepath(key string) string {
	h := sha256.Sum256([]byte(key))
	return filepath.Join(d.dir, hex.EncodeToString(h[:]))
}
