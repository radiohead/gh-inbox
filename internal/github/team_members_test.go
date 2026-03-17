package github

import (
	"errors"
	"fmt"
	"testing"

	"github.com/cli/go-gh/v2/pkg/api"
)

// mockCacher is a test double for Cacher.
type mockCacher struct {
	store    map[string][]byte
	getErr   error
	setErr   error
	getCalls []string
	setCalls []string
}

func newMockCacher() *mockCacher {
	return &mockCacher{store: make(map[string][]byte)}
}

func (m *mockCacher) Get(key string) ([]byte, bool, error) {
	m.getCalls = append(m.getCalls, key)
	if m.getErr != nil {
		return nil, false, m.getErr
	}
	data, found := m.store[key]
	return data, found, nil
}

func (m *mockCacher) Set(key string, data []byte) error {
	m.setCalls = append(m.setCalls, key)
	if m.setErr != nil {
		return m.setErr
	}
	m.store[key] = data
	return nil
}

// mockRESTDoer is a test double for restDoer.
type mockRESTDoer struct {
	getFunc func(path string, resp interface{}) error
}

func (m *mockRESTDoer) Get(path string, resp interface{}) error {
	return m.getFunc(path, resp)
}

func TestFetchCurrentUser(t *testing.T) {
	tests := []struct {
		name      string
		getFunc   func(path string, resp interface{}) error
		wantLogin string
		wantErr   bool
	}{
		{
			name: "happy path: returns login",
			getFunc: func(path string, resp interface{}) error {
				u := resp.(*userResponse)
				u.Login = "alice"
				return nil
			},
			wantLogin: "alice",
		},
		{
			name: "REST error: returns error",
			getFunc: func(path string, resp interface{}) error {
				return errors.New("REST API error")
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewClientWithDoers(nil, &mockRESTDoer{getFunc: tt.getFunc})
			login, err := client.FetchCurrentUser()

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if login != tt.wantLogin {
				t.Errorf("login = %q, want %q", login, tt.wantLogin)
			}
		})
	}
}

func TestFetchTeamMembers(t *testing.T) {
	tests := []struct {
		name    string
		getFunc func(path string, resp interface{}) error
		want    []TeamMember
		wantErr bool
	}{
		{
			name: "returns members on success",
			getFunc: func(path string, resp interface{}) error {
				members := resp.(*[]TeamMember)
				*members = []TeamMember{{Login: "alice"}, {Login: "bob"}}
				return nil
			},
			want: []TeamMember{{Login: "alice"}, {Login: "bob"}},
		},
		{
			name: "REST error: returns error",
			getFunc: func(path string, resp interface{}) error {
				return errors.New("team not found")
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewClientWithDoers(nil, &mockRESTDoer{getFunc: tt.getFunc})
			members, err := client.FetchTeamMembers("test-org", "backend")

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(members) != len(tt.want) {
				t.Fatalf("got %d members, want %d", len(members), len(tt.want))
			}
			for i, m := range members {
				if m.Login != tt.want[i].Login {
					t.Errorf("members[%d].Login = %q, want %q", i, m.Login, tt.want[i].Login)
				}
			}
		})
	}
}

func TestFetchChildTeams(t *testing.T) {
	tests := []struct {
		name    string
		getFunc func(path string, resp interface{}) error
		want    []ChildTeam
		wantErr bool
	}{
		{
			name: "returns child teams on success",
			getFunc: func(path string, resp interface{}) error {
				children := resp.(*[]ChildTeam)
				*children = []ChildTeam{{Slug: "frontend"}, {Slug: "backend"}}
				return nil
			},
			want: []ChildTeam{{Slug: "frontend"}, {Slug: "backend"}},
		},
		{
			name: "REST error: returns error",
			getFunc: func(path string, resp interface{}) error {
				return errors.New("team not found")
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewClientWithDoers(nil, &mockRESTDoer{getFunc: tt.getFunc})
			children, err := client.FetchChildTeams("test-org", "platform")

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(children) != len(tt.want) {
				t.Fatalf("got %d children, want %d", len(children), len(tt.want))
			}
			for i, c := range children {
				if c.Slug != tt.want[i].Slug {
					t.Errorf("children[%d].Slug = %q, want %q", i, c.Slug, tt.want[i].Slug)
				}
			}
		})
	}
}

func TestFetchIsOrgMember(t *testing.T) {
	tests := []struct {
		name    string
		getFunc func(path string, resp interface{}) error
		want    bool
		wantErr bool
	}{
		{
			name: "204 No Content: member",
			getFunc: func(path string, resp interface{}) error {
				return nil // 2xx success — member
			},
			want: true,
		},
		{
			name: "404 Not Found: not a member",
			getFunc: func(path string, resp interface{}) error {
				return &api.HTTPError{StatusCode: 404, Message: "Not Found"}
			},
			want: false,
		},
		{
			name: "other error: returns error",
			getFunc: func(path string, resp interface{}) error {
				return fmt.Errorf("network error")
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewClientWithDoers(nil, &mockRESTDoer{getFunc: tt.getFunc})
			got, err := client.FetchIsOrgMember("test-org", "alice")

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("FetchIsOrgMember = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFetchIsOrgMemberWithCache(t *testing.T) {
	const org = "myorg"
	const login = "alice"
	const cacheKey = "org-member:myorg:alice"

	t.Run("cache miss: calls API and writes true to cache", func(t *testing.T) {
		apiCalled := false
		rest := &mockRESTDoer{getFunc: func(path string, resp interface{}) error {
			apiCalled = true
			return nil // 204 member
		}}
		cacher := newMockCacher()
		client := NewClientWithDoers(nil, rest)
		client.cache = cacher

		got, err := client.FetchIsOrgMember(org, login)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !got {
			t.Errorf("FetchIsOrgMember = false, want true")
		}
		if !apiCalled {
			t.Error("expected API to be called on cache miss")
		}
		data, found := cacher.store[cacheKey]
		if !found {
			t.Fatal("expected cache entry to be written")
		}
		if string(data) != "true" {
			t.Errorf("cache value = %q, want %q", string(data), "true")
		}
	})

	t.Run("cache hit true: returns true without API call", func(t *testing.T) {
		apiCalled := false
		rest := &mockRESTDoer{getFunc: func(path string, resp interface{}) error {
			apiCalled = true
			return nil
		}}
		cacher := newMockCacher()
		cacher.store[cacheKey] = []byte("true")
		client := NewClientWithDoers(nil, rest)
		client.cache = cacher

		got, err := client.FetchIsOrgMember(org, login)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !got {
			t.Errorf("FetchIsOrgMember = false, want true")
		}
		if apiCalled {
			t.Error("expected API NOT to be called on cache hit")
		}
	})

	t.Run("cache hit false: returns false without API call", func(t *testing.T) {
		apiCalled := false
		rest := &mockRESTDoer{getFunc: func(path string, resp interface{}) error {
			apiCalled = true
			return nil
		}}
		cacher := newMockCacher()
		cacher.store["org-member:myorg:bob"] = []byte("false")
		client := NewClientWithDoers(nil, rest)
		client.cache = cacher

		got, err := client.FetchIsOrgMember(org, "bob")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got {
			t.Errorf("FetchIsOrgMember = true, want false")
		}
		if apiCalled {
			t.Error("expected API NOT to be called on cache hit")
		}
	})

	t.Run("cache Get error: falls through to API call", func(t *testing.T) {
		apiCalled := false
		rest := &mockRESTDoer{getFunc: func(path string, resp interface{}) error {
			apiCalled = true
			return nil // API succeeds
		}}
		cacher := newMockCacher()
		cacher.getErr = errors.New("cache read error")
		client := NewClientWithDoers(nil, rest)
		client.cache = cacher

		got, err := client.FetchIsOrgMember(org, login)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !got {
			t.Errorf("FetchIsOrgMember = false, want true")
		}
		if !apiCalled {
			t.Error("expected API to be called when cache Get returns error")
		}
	})

	t.Run("cache Set error: returns correct result, ignores Set error", func(t *testing.T) {
		rest := &mockRESTDoer{getFunc: func(path string, resp interface{}) error {
			return nil // 204 member
		}}
		cacher := newMockCacher()
		cacher.setErr = errors.New("cache write error")
		client := NewClientWithDoers(nil, rest)
		client.cache = cacher

		got, err := client.FetchIsOrgMember(org, login)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !got {
			t.Errorf("FetchIsOrgMember = false, want true")
		}
	})

	t.Run("API error: returns false and error, does not write to cache", func(t *testing.T) {
		rest := &mockRESTDoer{getFunc: func(path string, resp interface{}) error {
			return fmt.Errorf("network error")
		}}
		cacher := newMockCacher()
		client := NewClientWithDoers(nil, rest)
		client.cache = cacher

		got, err := client.FetchIsOrgMember(org, login)
		if err == nil {
			t.Error("expected error, got nil")
		}
		if got {
			t.Errorf("FetchIsOrgMember = true, want false on API error")
		}
		if len(cacher.setCalls) > 0 {
			t.Error("expected no cache writes on API error")
		}
	})
}
