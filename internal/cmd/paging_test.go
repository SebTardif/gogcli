package cmd

import (
	"errors"
	"strings"
	"testing"
)

func TestCollectAllPages(t *testing.T) {
	t.Parallel()

	t.Run("collects successive pages", func(t *testing.T) {
		t.Parallel()
		pages := map[string]struct {
			items []string
			next  string
		}{
			"":   {items: []string{"a"}, next: "p2"},
			"p2": {items: []string{"b", "c"}, next: ""},
		}
		got, err := collectAllPages("", func(pageToken string) ([]string, string, error) {
			page, ok := pages[pageToken]
			if !ok {
				t.Fatalf("unexpected page token %q", pageToken)
			}
			return page.items, page.next, nil
		})
		if err != nil {
			t.Fatalf("err = %v", err)
		}
		if strings.Join(got, ",") != "a,b,c" {
			t.Fatalf("got %v, want [a b c]", got)
		}
	})

	t.Run("rejects a repeated page token", func(t *testing.T) {
		t.Parallel()
		var calls int
		_, err := collectAllPages("", func(string) ([]string, string, error) {
			calls++
			return []string{"item"}, "stuck", nil
		})
		if err == nil || !strings.Contains(err.Error(), `repeated page token "stuck"`) {
			t.Fatalf("err = %v", err)
		}
		if calls != 2 {
			t.Fatalf("calls = %d, want 2", calls)
		}
	})

	t.Run("returns fetch errors", func(t *testing.T) {
		t.Parallel()
		want := errors.New("boom")
		_, err := collectAllPages("", func(string) ([]string, string, error) {
			return nil, "", want
		})
		if !errors.Is(err, want) {
			t.Fatalf("err = %v, want %v", err, want)
		}
	})
}
