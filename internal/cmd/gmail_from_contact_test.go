package cmd

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"google.golang.org/api/people/v1"
)

func TestBuildGmailFromEmailsQuery(t *testing.T) {
	if got := buildGmailFromEmailsQuery([]string{"a@example.com"}); got != "from:a@example.com" {
		t.Fatalf("single = %q", got)
	}
	if got := buildGmailFromEmailsQuery([]string{"a@example.com", "b@example.com"}); got != "from:(a@example.com OR b@example.com)" {
		t.Fatalf("multi = %q", got)
	}
}

func TestSelectGmailFromContactPeoplePrefersExactMatch(t *testing.T) {
	resp := &people.SearchResponse{Results: []*people.SearchResult{
		{Person: &people.Person{Names: []*people.Name{{DisplayName: "Alice A"}}, EmailAddresses: []*people.EmailAddress{{Value: "alice@example.com"}}}},
		{Person: &people.Person{Names: []*people.Name{{DisplayName: "Alice B"}}, EmailAddresses: []*people.EmailAddress{{Value: "b@example.com"}}}},
	}}
	got := selectGmailFromContactPeople("alice@example.com", resp)
	if len(got) != 1 || primaryName(got[0]) != "Alice A" {
		t.Fatalf("unexpected selection: %#v", got)
	}
}

func TestAllContactEmailsDedupes(t *testing.T) {
	got := allContactEmails(&people.Person{EmailAddresses: []*people.EmailAddress{
		{Value: "A@example.com"},
		{Value: "a@example.com"},
		{Value: "b@example.com"},
	}})
	if len(got) != 2 || got[0] != "A@example.com" || got[1] != "b@example.com" {
		t.Fatalf("emails = %#v", got)
	}
}

func TestGmailFromContactQuery_WarmsContactsSearchCache(t *testing.T) {
	var queries []string
	svc := newPeopleSearchTestService(t, "people:searchContacts", "people/c1", "Alice", "alice@example.com", &queries)

	got, err := gmailFromContactQuery(withPeopleContactsTestService(context.Background(), svc), "a@b.com", "Alice")
	if err != nil {
		t.Fatalf("gmailFromContactQuery: %v", err)
	}
	if got != "from:alice@example.com" {
		t.Fatalf("query = %q", got)
	}
	if got, want := strings.Join(queries, ","), ",Alice"; got != want {
		t.Fatalf("search queries = %q, want %q", got, want)
	}
}

func TestGmailFromContactFallbackRejectsRepeatedPageToken(t *testing.T) {
	var listCalls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.Contains(r.URL.Path, "people:searchContacts") && r.Method == http.MethodGet:
			_ = json.NewEncoder(w).Encode(map[string]any{"results": []map[string]any{}})
		case strings.Contains(r.URL.Path, "/people/me/connections") && r.Method == http.MethodGet:
			listCalls++
			if listCalls > 8 {
				t.Fatalf("repeated page token looped: %d list calls", listCalls)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"connections": []map[string]any{
					{
						"resourceName":   "people/c1",
						"names":          []map[string]any{{"displayName": "Ada"}},
						"emailAddresses": []map[string]any{{"value": "ada@example.com"}},
					},
				},
				"nextPageToken": "stuck",
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	svc := newPeopleServiceFromServer(t, srv)
	_, err := gmailFromContactQuery(withPeopleContactsTestService(context.Background(), svc), "a@b.com", "Ada")
	if err == nil || !strings.Contains(err.Error(), "repeated page token") {
		t.Fatalf("err = %v after %d list calls", err, listCalls)
	}
	t.Logf("err = %v after %d list calls", err, listCalls)
}
