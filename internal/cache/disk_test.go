package cache

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDiskCacher(t *testing.T) {
	tests := []struct {
		name      string
		setup     func(t *testing.T, dc *DiskCacher)
		key       string
		wantData  []byte
		wantFound bool
		wantErr   bool
	}{
		{
			name:      "cache miss when file absent",
			setup:     func(t *testing.T, dc *DiskCacher) {},
			key:       "nonexistent",
			wantData:  nil,
			wantFound: false,
		},
		{
			name: "cache hit when file fresh",
			setup: func(t *testing.T, dc *DiskCacher) {
				t.Helper()
				if err := dc.Set("hit-key", []byte(`{"ok":true}`)); err != nil {
					t.Fatal(err)
				}
			},
			key:       "hit-key",
			wantData:  []byte(`{"ok":true}`),
			wantFound: true,
		},
		{
			name: "stale entry treated as miss",
			setup: func(t *testing.T, dc *DiskCacher) {
				t.Helper()
				if err := dc.Set("stale-key", []byte("old")); err != nil {
					t.Fatal(err)
				}
				// Set mtime to 2 hours ago (TTL is 1 hour in tests).
				path := dc.filepath("stale-key")
				past := time.Now().Add(-2 * time.Hour)
				if err := os.Chtimes(path, past, past); err != nil {
					t.Fatal(err)
				}
			},
			key:       "stale-key",
			wantData:  nil,
			wantFound: false,
		},
		{
			name: "overwrite returns latest value",
			setup: func(t *testing.T, dc *DiskCacher) {
				t.Helper()
				if err := dc.Set("ow-key", []byte("v1")); err != nil {
					t.Fatal(err)
				}
				if err := dc.Set("ow-key", []byte("v2")); err != nil {
					t.Fatal(err)
				}
			},
			key:       "ow-key",
			wantData:  []byte("v2"),
			wantFound: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dc, err := NewDiskCacher(t.TempDir(), 1*time.Hour)
			if err != nil {
				t.Fatal(err)
			}
			tt.setup(t, dc)

			data, found, err := dc.Get(tt.key)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Get() error = %v, wantErr %v", err, tt.wantErr)
			}
			if found != tt.wantFound {
				t.Fatalf("Get() found = %v, want %v", found, tt.wantFound)
			}
			if string(data) != string(tt.wantData) {
				t.Fatalf("Get() data = %q, want %q", data, tt.wantData)
			}
		})
	}
}

func TestDiskCacher_SetWriteFailure(t *testing.T) {
	// Use a non-existent nested path that cannot be created as a file.
	badDir := filepath.Join(t.TempDir(), "no-such-dir", "nested")
	dc := &DiskCacher{dir: badDir, ttl: time.Hour}

	err := dc.Set("key", []byte("data"))
	if err == nil {
		t.Fatal("expected Set to return error for non-writable directory")
	}

	// Subsequent Get should return a miss, not an error.
	data, found, getErr := dc.Get("key")
	if found {
		t.Fatal("expected Get to return miss after write failure")
	}
	if data != nil {
		t.Fatalf("expected nil data, got %q", data)
	}
	if getErr != nil {
		t.Fatalf("expected no error from Get, got %v", getErr)
	}
}

func TestDiskCacher_DifferentKeysIndependent(t *testing.T) {
	dc, err := NewDiskCacher(t.TempDir(), time.Hour)
	if err != nil {
		t.Fatal(err)
	}

	if err := dc.Set("alpha", []byte("a-data")); err != nil {
		t.Fatal(err)
	}
	if err := dc.Set("beta", []byte("b-data")); err != nil {
		t.Fatal(err)
	}

	aData, aFound, _ := dc.Get("alpha")
	bData, bFound, _ := dc.Get("beta")

	if !aFound || string(aData) != "a-data" {
		t.Fatalf("alpha: found=%v data=%q", aFound, aData)
	}
	if !bFound || string(bData) != "b-data" {
		t.Fatalf("beta: found=%v data=%q", bFound, bData)
	}
}
