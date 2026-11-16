package reconciler

import "strings"

// GlobalAliases defines bidirectional equivalence groups for ubiquitous parameter names.
var GlobalAliases = [][]string{
	{"userid", "user_id", "user", "uid", "id", "customerid", "customer_id", "clientid", "client_id"},
	{"useremail", "user_email", "email", "mail", "contact_email"},
	{"query", "q", "search", "search_term", "keyword", "term"},
	{"pagesize", "page_size", "limit", "per_page", "count", "max_results", "size"},
	{"pagenumber", "page_number", "page", "page_index", "offset"},
	{"createddate", "created_date", "created_at", "createdAt", "timestamp", "date", "since", "start_date"},
	{"amountcents", "amount_cents", "amount", "total", "cents", "price", "cost"},
	{"issuetitle", "issue_title", "title", "subject", "summary", "name", "headline"},
	{"issuebody", "issue_body", "body", "description", "content", "desc", "details"},
	{"messagetext", "message_text", "message", "text", "body", "content", "msg"},
	{"channelid", "channel_id", "channel", "target_channel", "room", "room_id"},
	{"isactive", "is_active", "active", "enabled", "status"},
	{"isurgent", "is_urgent", "urgent", "priority", "critical"},
	{"capturenow", "capture_now", "capture", "auto_capture"},
	{"assignees", "assignee_list", "assigned_to", "assignee", "user_list"},
	{"labels", "label_list", "tags", "tag_list", "categories"},
}

// FindGlobalAlias maps an input key to any candidate in the schema that shares an alias equivalence group.
func FindGlobalAlias(inputKey string, schemaCandidates []string) (string, bool) {
	inputFolded := FoldKey(inputKey)

	// Identify which groups match inputFolded
	for _, group := range GlobalAliases {
		matchedGroup := false
		for _, member := range group {
			if FoldKey(member) == inputFolded {
				matchedGroup = true
				break
			}
		}

		if matchedGroup {
			// Find if any schema candidate is in this group
			for _, cand := range schemaCandidates {
				candFolded := FoldKey(cand)
				for _, member := range group {
					if FoldKey(member) == candFolded {
						return cand, true
					}
				}
			}
		}
	}

	return "", false
}

// FindExplicitAlias checks if input matches any explicit alias declared in a property's schema.
func FindExplicitAlias(inputKey string, aliases []string) bool {
	inputFolded := FoldKey(inputKey)
	for _, alias := range aliases {
		if FoldKey(alias) == inputFolded || strings.EqualFold(alias, inputKey) {
			return true
		}
	}
	return false
}
