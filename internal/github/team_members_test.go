package github

import (
	"errors"
	"testing"
)

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

func TestIsTeamMember(t *testing.T) {
	buildMembersResponse := func(logins []string) func(path string, resp interface{}) error {
		return func(path string, resp interface{}) error {
			members := resp.(*[]teamMemberResponse)
			for _, login := range logins {
				*members = append(*members, teamMemberResponse{Login: login})
			}
			return nil
		}
	}

	tests := []struct {
		name    string
		getFunc func(path string, resp interface{}) error
		org     string
		slug    string
		login   string
		want    bool
	}{
		{
			name:      "cache miss, user IS member",
			getFunc:   buildMembersResponse([]string{"alice", "bob"}),
			org:       "test-org",
			slug:      "backend",
			login: "alice",
			want:  true,
		},
		{
			name:    "cache miss, user is NOT member",
			getFunc: buildMembersResponse([]string{"carol", "dave"}),
			org:     "test-org",
			slug:    "backend",
			login:   "alice",
			want:    false,
		},
		{
			name: "REST error: fail-open returns true",
			getFunc: func(path string, resp interface{}) error {
				return errors.New("team not found")
			},
			org:   "test-org",
			slug:  "backend",
			login: "alice",
			want:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewClientWithDoers(nil, &mockRESTDoer{getFunc: tt.getFunc})
			got := client.IsTeamMember(tt.org, tt.slug, tt.login)
			if got != tt.want {
				t.Errorf("IsTeamMember = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsTeamMemberCacheHit(t *testing.T) {
	callCount := 0
	getFunc := func(path string, resp interface{}) error {
		callCount++
		members := resp.(*[]teamMemberResponse)
		*members = append(*members, teamMemberResponse{Login: "alice"})
		return nil
	}

	client := NewClientWithDoers(nil, &mockRESTDoer{getFunc: getFunc})

	// First call — cache miss, should fetch from REST.
	got1 := client.IsTeamMember("test-org", "backend", "alice")
	if !got1 {
		t.Errorf("first call: IsTeamMember = false, want true")
	}

	// Second call with same org/slug — should use cache, not call REST again.
	got2 := client.IsTeamMember("test-org", "backend", "alice")
	if !got2 {
		t.Errorf("second call: IsTeamMember = false, want true")
	}

	if callCount != 1 {
		t.Errorf("REST called %d times, want 1 (cache should prevent second call)", callCount)
	}
}
