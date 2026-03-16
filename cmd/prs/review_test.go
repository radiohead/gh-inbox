package prs

import (
	"testing"

	"github.com/radiohead/gh-inbox/internal/service"
)

func TestResolveFilter(t *testing.T) {
	openStatus := service.ReviewStatusSet{service.ReviewStatusOpen: true}

	tests := []struct {
		name         string
		filter       string
		filterType   string
		filterSource string
		filterStatus string
		wantCriteria service.FilterCriteria
		wantErr      bool
	}{
		// Preset mode — default ReviewStatuses per preset
		{
			name: "no flags defaults to open status",
			wantCriteria: service.FilterCriteria{
				ReviewStatuses: openStatus,
			},
		},
		{
			name:         "filter=all — nil ReviewStatuses",
			filter:       "all",
			wantCriteria: service.FilterCriteria{},
		},
		{
			name:   "filter=focus — open ReviewStatuses",
			filter: "focus",
			wantCriteria: service.FilterCriteria{
				ReviewTypes:    service.ReviewTypeSet{service.ReviewTypeDirect: true, service.ReviewTypeCodeowner: true},
				AuthorSources:  service.AuthorSourceSet{service.AuthorSourceTeam: true},
				ReviewStatuses: openStatus,
			},
		},
		{
			name:   "filter=nearby — open ReviewStatuses",
			filter: "nearby",
			wantCriteria: service.FilterCriteria{
				ReviewTypes:    service.ReviewTypeSet{service.ReviewTypeDirect: true, service.ReviewTypeCodeowner: true},
				AuthorSources:  service.AuthorSourceSet{service.AuthorSourceTeam: true, service.AuthorSourceGroup: true},
				ReviewStatuses: openStatus,
			},
		},
		{
			name:   "filter=org — nil ReviewStatuses",
			filter: "org",
			wantCriteria: service.FilterCriteria{
				AuthorSources: service.AuthorSourceSet{service.AuthorSourceTeam: true, service.AuthorSourceGroup: true, service.AuthorSourceOrg: true},
			},
		},
		{
			name:    "filter=invalid",
			filter:  "unknown",
			wantErr: true,
		},
		// Granular mode — default open status
		{
			name:       "filter-type=direct — defaults to open status",
			filterType: "direct",
			wantCriteria: service.FilterCriteria{
				ReviewTypes:    service.ReviewTypeSet{service.ReviewTypeDirect: true},
				ReviewStatuses: openStatus,
			},
		},
		{
			name:       "filter-type=codeowner",
			filterType: "codeowner",
			wantCriteria: service.FilterCriteria{
				ReviewTypes:    service.ReviewTypeSet{service.ReviewTypeCodeowner: true},
				ReviewStatuses: openStatus,
			},
		},
		{
			name:       "filter-type=team",
			filterType: "team",
			wantCriteria: service.FilterCriteria{
				ReviewTypes:    service.ReviewTypeSet{service.ReviewTypeTeam: true},
				ReviewStatuses: openStatus,
			},
		},
		{
			name:         "filter-source=team (lowercase normalised)",
			filterSource: "team",
			wantCriteria: service.FilterCriteria{
				AuthorSources:  service.AuthorSourceSet{service.AuthorSourceTeam: true},
				ReviewStatuses: openStatus,
			},
		},
		{
			name:         "filter-source=TEAM (uppercase)",
			filterSource: "TEAM",
			wantCriteria: service.FilterCriteria{
				AuthorSources:  service.AuthorSourceSet{service.AuthorSourceTeam: true},
				ReviewStatuses: openStatus,
			},
		},
		{
			name:         "filter-source=group",
			filterSource: "group",
			wantCriteria: service.FilterCriteria{
				AuthorSources:  service.AuthorSourceSet{service.AuthorSourceGroup: true},
				ReviewStatuses: openStatus,
			},
		},
		{
			name:         "filter-source=org",
			filterSource: "org",
			wantCriteria: service.FilterCriteria{
				AuthorSources:  service.AuthorSourceSet{service.AuthorSourceOrg: true},
				ReviewStatuses: openStatus,
			},
		},
		{
			name:         "filter-source=other",
			filterSource: "other",
			wantCriteria: service.FilterCriteria{
				AuthorSources:  service.AuthorSourceSet{service.AuthorSourceOther: true},
				ReviewStatuses: openStatus,
			},
		},
		{
			name:         "filter-type=direct filter-source=team (combined)",
			filterType:   "direct",
			filterSource: "team",
			wantCriteria: service.FilterCriteria{
				ReviewTypes:    service.ReviewTypeSet{service.ReviewTypeDirect: true},
				AuthorSources:  service.AuthorSourceSet{service.AuthorSourceTeam: true},
				ReviewStatuses: openStatus,
			},
		},
		// Errors
		{
			name:       "filter-type=invalid",
			filterType: "invalid",
			wantErr:    true,
		},
		{
			name:         "filter-source=invalid",
			filterSource: "invalid",
			wantErr:      true,
		},
		// --filter-status flag
		{
			name:         "filter-status=open",
			filterStatus: "open",
			wantCriteria: service.FilterCriteria{
				ReviewStatuses: openStatus,
			},
		},
		{
			name:         "filter-status=in_review",
			filterStatus: "in_review",
			wantCriteria: service.FilterCriteria{
				ReviewStatuses: service.ReviewStatusSet{service.ReviewStatusInReview: true},
			},
		},
		{
			name:         "filter-status=approved",
			filterStatus: "approved",
			wantCriteria: service.FilterCriteria{
				ReviewStatuses: service.ReviewStatusSet{service.ReviewStatusApproved: true},
			},
		},
		{
			name:         "filter-status=all — nil ReviewStatuses",
			filterStatus: "all",
			wantCriteria: service.FilterCriteria{},
		},
		{
			name:         "filter-status overrides focus preset default",
			filter:       "focus",
			filterStatus: "in_review",
			wantCriteria: service.FilterCriteria{
				ReviewTypes:    service.ReviewTypeSet{service.ReviewTypeDirect: true, service.ReviewTypeCodeowner: true},
				AuthorSources:  service.AuthorSourceSet{service.AuthorSourceTeam: true},
				ReviewStatuses: service.ReviewStatusSet{service.ReviewStatusInReview: true},
			},
		},
		{
			name:         "filter-status=all overrides focus preset to nil",
			filter:       "focus",
			filterStatus: "all",
			wantCriteria: service.FilterCriteria{
				ReviewTypes:   service.ReviewTypeSet{service.ReviewTypeDirect: true, service.ReviewTypeCodeowner: true},
				AuthorSources: service.AuthorSourceSet{service.AuthorSourceTeam: true},
			},
		},
		{
			name:         "filter-status overrides org preset nil default",
			filter:       "org",
			filterStatus: "open",
			wantCriteria: service.FilterCriteria{
				AuthorSources:  service.AuthorSourceSet{service.AuthorSourceTeam: true, service.AuthorSourceGroup: true, service.AuthorSourceOrg: true},
				ReviewStatuses: openStatus,
			},
		},
		{
			name:         "filter-status=invalid",
			filterStatus: "invalid",
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := resolveFilter(tt.filter, tt.filterType, tt.filterSource, tt.filterStatus)
			if tt.wantErr {
				if err == nil {
					t.Errorf("resolveFilter() expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("resolveFilter() unexpected error: %v", err)
			}
			if len(got.ReviewTypes) != len(tt.wantCriteria.ReviewTypes) {
				t.Errorf("ReviewTypes = %v, want %v", got.ReviewTypes, tt.wantCriteria.ReviewTypes)
			}
			for rt := range tt.wantCriteria.ReviewTypes {
				if !got.ReviewTypes[rt] {
					t.Errorf("ReviewTypes missing %q", rt)
				}
			}
			if len(got.AuthorSources) != len(tt.wantCriteria.AuthorSources) {
				t.Errorf("AuthorSources = %v, want %v", got.AuthorSources, tt.wantCriteria.AuthorSources)
			}
			for as := range tt.wantCriteria.AuthorSources {
				if !got.AuthorSources[as] {
					t.Errorf("AuthorSources missing %q", as)
				}
			}
			if len(got.ReviewStatuses) != len(tt.wantCriteria.ReviewStatuses) {
				t.Errorf("ReviewStatuses = %v, want %v", got.ReviewStatuses, tt.wantCriteria.ReviewStatuses)
			}
			for rs := range tt.wantCriteria.ReviewStatuses {
				if !got.ReviewStatuses[rs] {
					t.Errorf("ReviewStatuses missing %q", rs)
				}
			}
		})
	}
}
