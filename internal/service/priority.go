package service

// Priority returns an integer priority for a classified PR.
// Lower values indicate higher priority (closer to the user's immediate concern).
// Ordering: source proximity (primary), review type weight (secondary).
func Priority(cp ClassifiedPR) int {
	return authorSourceWeight(cp.AuthorSource)*10 + reviewTypeWeight(cp.ReviewType)
}

// authorSourceWeight returns the sort weight for an AuthorSource.
// TEAM=0 (closest), GROUP=1, ORG=2, OTHER/unknown=3.
func authorSourceWeight(s AuthorSource) int {
	switch s {
	case AuthorSourceTeam:
		return 0
	case AuthorSourceGroup:
		return 1
	case AuthorSourceOrg:
		return 2
	default: // OTHER or empty
		return 3
	}
}

// reviewTypeWeight returns the sort weight for a ReviewType.
// direct=0 (most urgent), codeowner=1, team=2, unknown=3.
func reviewTypeWeight(t ReviewType) int {
	switch t {
	case ReviewTypeDirect:
		return 0
	case ReviewTypeCodeowner:
		return 1
	case ReviewTypeTeam:
		return 2
	default: // empty or unknown
		return 3
	}
}
