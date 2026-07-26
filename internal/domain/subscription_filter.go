package domain

import "strings"

// FilterSubscriptionsByName returns subscriptions whose DisplayName contains
// the query string (case-insensitive).  An empty query returns all subscriptions.
func FilterSubscriptionsByName(subs []Subscription, query string) []Subscription {
	if query == "" {
		return subs
	}

	lower := strings.ToLower(query)
	var filtered []Subscription
	for _, s := range subs {
		if strings.Contains(strings.ToLower(s.DisplayName), lower) {
			filtered = append(filtered, s)
		}
	}
	return filtered
}