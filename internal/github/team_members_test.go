package github

import (
	"errors"
	"fmt"
	"testing"

	"github.com/cli/go-gh/v2/pkg/api"
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
