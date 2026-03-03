package service

import "testing"

func TestPriority(t *testing.T) {
	tests := []struct {
		name   string
		source AuthorSource
		rtype  ReviewType
		want   int
	}{
		// TEAM bucket
		{name: "TEAM+direct", source: AuthorSourceTeam, rtype: ReviewTypeDirect, want: 0},
		{name: "TEAM+codeowner", source: AuthorSourceTeam, rtype: ReviewTypeCodeowner, want: 1},
		{name: "TEAM+team", source: AuthorSourceTeam, rtype: ReviewTypeTeam, want: 2},
		// GROUP bucket
		{name: "GROUP+direct", source: AuthorSourceGroup, rtype: ReviewTypeDirect, want: 10},
		{name: "GROUP+codeowner", source: AuthorSourceGroup, rtype: ReviewTypeCodeowner, want: 11},
		{name: "GROUP+team", source: AuthorSourceGroup, rtype: ReviewTypeTeam, want: 12},
		// ORG bucket
		{name: "ORG+direct", source: AuthorSourceOrg, rtype: ReviewTypeDirect, want: 20},
		{name: "ORG+codeowner", source: AuthorSourceOrg, rtype: ReviewTypeCodeowner, want: 21},
		{name: "ORG+team", source: AuthorSourceOrg, rtype: ReviewTypeTeam, want: 22},
		// OTHER bucket
		{name: "OTHER+direct", source: AuthorSourceOther, rtype: ReviewTypeDirect, want: 30},
		{name: "OTHER+codeowner", source: AuthorSourceOther, rtype: ReviewTypeCodeowner, want: 31},
		{name: "OTHER+team", source: AuthorSourceOther, rtype: ReviewTypeTeam, want: 32},
		// Empty/unknown values map to max weight
		{name: "empty source+empty type", source: "", rtype: "", want: 33},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cp := ClassifiedPR{ReviewType: tt.rtype, AuthorSource: tt.source}
			got := Priority(cp)
			if got != tt.want {
				t.Errorf("Priority() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestPriorityOrdering(t *testing.T) {
	// Verify the natural ordering: TEAM+direct is strictly better than GROUP+direct,
	// GROUP+direct is better than GROUP+codeowner, etc.
	teamDirect := ClassifiedPR{ReviewType: ReviewTypeDirect, AuthorSource: AuthorSourceTeam}
	teamCodeowner := ClassifiedPR{ReviewType: ReviewTypeCodeowner, AuthorSource: AuthorSourceTeam}
	groupDirect := ClassifiedPR{ReviewType: ReviewTypeDirect, AuthorSource: AuthorSourceGroup}
	orgTeam := ClassifiedPR{ReviewType: ReviewTypeTeam, AuthorSource: AuthorSourceOrg}
	other := ClassifiedPR{ReviewType: ReviewTypeCodeowner, AuthorSource: AuthorSourceOther}

	cases := []struct {
		higher ClassifiedPR
		lower  ClassifiedPR
	}{
		{teamDirect, teamCodeowner},   // same source, direct beats codeowner
		{teamDirect, groupDirect},     // same type, TEAM beats GROUP
		{teamCodeowner, groupDirect},  // TEAM+codeowner still beats GROUP+direct
		{groupDirect, orgTeam},        // GROUP beats ORG regardless of type
		{orgTeam, other},              // ORG beats OTHER
	}

	for _, c := range cases {
		ph := Priority(c.higher)
		pl := Priority(c.lower)
		if ph >= pl {
			t.Errorf("expected Priority(%+v)=%d < Priority(%+v)=%d", c.higher, ph, c.lower, pl)
		}
	}
}
