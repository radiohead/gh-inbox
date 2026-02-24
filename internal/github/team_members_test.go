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
