package main

var mockStacks = []stack{
	{
		ID: "checkout-retries", Title: "Checkout retry handling", Repository: "acme/storefront", Owner: "Alex Morgan", Initials: "AM", Mine: true, Team: true, Updated: "12m ago",
		PRs: []pullRequest{
			{Number: 482, Title: "Add retry policy types", Branch: "checkout/retry-types", Author: "octocat", Review: "approved", Checks: "passing", Comments: 3, Additions: 184, Deletions: 42, Updated: "2d ago", MergeTarget: "main"},
			{Number: 486, Title: "Retry transient payment failures", Branch: "checkout/retry-payments", Author: "octocat", Review: "changes", Checks: "passing", Comments: 8, Additions: 312, Deletions: 91, Updated: "1d ago", MergeTarget: "#482"},
			{Number: 490, Title: "Show retry state in order history", Branch: "checkout/retry-status", Author: "octocat", Review: "waiting", Checks: "running", Comments: 1, Additions: 97, Deletions: 12, Updated: "3h ago", MergeTarget: "#486"},
		},
	},
	{
		ID: "search-metrics", Title: "Search indexing observability", Repository: "acme/search", Owner: "Sam Rivera", Initials: "SR", Assigned: true, Team: true, Updated: "34m ago",
		PRs: []pullRequest{
			{Number: 201, Title: "Record indexing latency", Branch: "search/index-latency", Author: "srivera", Review: "approved", Checks: "passing", Comments: 4, Additions: 226, Deletions: 18, Updated: "3d ago", MergeTarget: "main", Queued: true, QueuePosition: 2, QueueState: "awaiting_checks", QueueETA: 240},
			{Number: 205, Title: "Add index health dashboard", Branch: "search/health-dashboard", Author: "srivera", Review: "waiting", Checks: "failing", Comments: 6, Additions: 488, Deletions: 63, Updated: "1d ago", MergeTarget: "#201"},
		},
	},
	{
		ID: "session-security", Title: "Session token rotation", Repository: "acme/api", Owner: "Jordan Lee", Initials: "JL", Assigned: true, Team: true, Updated: "1h ago",
		PRs: []pullRequest{
			{Number: 364, Title: "Define token rotation contracts", Branch: "sessions/rotation-contracts", Author: "jlee", Review: "approved", Checks: "passing", Comments: 2, Additions: 143, Deletions: 9, Updated: "5d ago", MergeTarget: "main"},
			{Number: 371, Title: "Persist token rotation state", Branch: "sessions/rotation-state", Author: "jlee", Review: "approved", Checks: "passing", Comments: 11, Additions: 621, Deletions: 77, Updated: "4d ago", MergeTarget: "#364"},
			{Number: 378, Title: "Show active sessions in settings", Branch: "sessions/settings", Author: "jlee", Review: "waiting", Checks: "passing", Comments: 5, Additions: 354, Deletions: 44, Updated: "2d ago", MergeTarget: "#371"},
			{Number: 388, Title: "Add session revocation flow", Branch: "sessions/revocation", Author: "jlee", Review: "draft", Checks: "running", Comments: 0, Additions: 502, Deletions: 103, Updated: "8h ago", MergeTarget: "#378"},
		},
	},
	{
		ID: "mobile-release", Title: "Mobile release metadata", Repository: "acme/mobile", Owner: "Taylor Kim", Initials: "TK", Team: true, Updated: "3h ago",
		PRs: []pullRequest{
			{Number: 151, Title: "Generate release metadata", Branch: "release/metadata", Author: "tkim", Review: "approved", Checks: "passing", Comments: 7, Additions: 277, Deletions: 125, Updated: "6d ago", MergeTarget: "main"},
			{Number: 169, Title: "Display release notes in app", Branch: "release/notes", Author: "tkim", Review: "approved", Checks: "passing", Comments: 3, Additions: 198, Deletions: 24, Updated: "4d ago", MergeTarget: "#151"},
		},
	},
	{
		ID: "audit-export", Title: "Audit event export", Repository: "acme/api", Owner: "Casey Nguyen", Initials: "CN", Assigned: true, Team: true, Updated: "5h ago",
		PRs: []pullRequest{
			{Number: 292, Title: "Define audit export schema", Branch: "audit/export-schema", Author: "cnguyen", Review: "approved", Checks: "passing", Comments: 4, Additions: 151, Deletions: 22, Updated: "5d ago", MergeTarget: "main"},
			{Number: 296, Title: "Stream audit events to export", Branch: "audit/export-stream", Author: "cnguyen", Review: "waiting", Checks: "passing", Comments: 2, Additions: 289, Deletions: 31, Updated: "2d ago", MergeTarget: "#292"},
		},
	},
}
