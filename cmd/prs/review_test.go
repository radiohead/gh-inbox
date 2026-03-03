package prs

import (
	"testing"

	"github.com/radiohead/gh-inbox/internal/service"
)

func TestResolveFilter(t *testing.T) {
	tests := []struct {
		name         string
		filter       string
		filterType   string
		filterSource string
		wantCriteria service.FilterCriteria
		wantErr      bool
	}{
		// Preset mode
		{
			name:         "no flags defaults to all",
			wantCriteria: service.FilterCriteria{},
		},
		{
			name:         "filter=all",
			filter:       "all",
			wantCriteria: service.FilterCriteria{},
		},
		{
			name:   "filter=focus",
			filter: "focus",
			wantCriteria: service.FilterCriteria{
				ReviewTypes:   service.ReviewTypeSet{service.ReviewTypeDirect: true, service.ReviewTypeCodeowner: true},
				AuthorSources: service.AuthorSourceSet{service.AuthorSourceTeam: true},
			},
		},
		{
			name:   "filter=nearby",
			filter: "nearby",
			wantCriteria: service.FilterCriteria{
				ReviewTypes:   service.ReviewTypeSet{service.ReviewTypeDirect: true, service.ReviewTypeCodeowner: true},
				AuthorSources: service.AuthorSourceSet{service.AuthorSourceTeam: true, service.AuthorSourceGroup: true},
			},
		},
		{
			name:   "filter=org",
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
		// Granular mode
		{
			name:         "filter-type=direct",
			filterType:   "direct",
			wantCriteria: service.FilterCriteria{ReviewTypes: service.ReviewTypeSet{service.ReviewTypeDirect: true}},
		},
		{
			name:         "filter-type=codeowner",
			filterType:   "codeowner",
			wantCriteria: service.FilterCriteria{ReviewTypes: service.ReviewTypeSet{service.ReviewTypeCodeowner: true}},
		},
		{
			name:         "filter-type=team",
			filterType:   "team",
			wantCriteria: service.FilterCriteria{ReviewTypes: service.ReviewTypeSet{service.ReviewTypeTeam: true}},
		},
		{
			name:         "filter-source=team (lowercase normalised)",
			filterSource: "team",
			wantCriteria: service.FilterCriteria{AuthorSources: service.AuthorSourceSet{service.AuthorSourceTeam: true}},
		},
		{
			name:         "filter-source=TEAM (uppercase)",
			filterSource: "TEAM",
			wantCriteria: service.FilterCriteria{AuthorSources: service.AuthorSourceSet{service.AuthorSourceTeam: true}},
		},
		{
			name:         "filter-source=group",
			filterSource: "group",
			wantCriteria: service.FilterCriteria{AuthorSources: service.AuthorSourceSet{service.AuthorSourceGroup: true}},
		},
		{
			name:         "filter-source=org",
			filterSource: "org",
			wantCriteria: service.FilterCriteria{AuthorSources: service.AuthorSourceSet{service.AuthorSourceOrg: true}},
		},
		{
			name:         "filter-source=other",
			filterSource: "other",
			wantCriteria: service.FilterCriteria{AuthorSources: service.AuthorSourceSet{service.AuthorSourceOther: true}},
		},
		{
			name:       "filter-type=direct filter-source=team (combined)",
			filterType: "direct",
			filterSource: "team",
			wantCriteria: service.FilterCriteria{
				ReviewTypes:   service.ReviewTypeSet{service.ReviewTypeDirect: true},
				AuthorSources: service.AuthorSourceSet{service.AuthorSourceTeam: true},
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := resolveFilter(tt.filter, tt.filterType, tt.filterSource)
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
		})
	}
}
