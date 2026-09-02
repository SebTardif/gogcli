package cmd

import (
	"errors"
	"strings"
	"testing"
)

func TestCollectAllPages(t *testing.T) {
	t.Parallel()

	t.Run("empty next token ends", func(t *testing.T) {
		t.Parallel()
		var calls int
		got, err := collectAllPages("", func(pageToken string) ([]string, string, error) {
			calls++
			if pageToken != "" {
				t.Fatalf("pageToken = %q, want empty", pageToken)
			}
			return []string{"a"}, "", nil
		})
		if err != nil {
			t.Fatalf("err = %v", err)
		}
		if calls != 1 {
			t.Fatalf("calls = %d, want 1", calls)
		}
		if len(got) != 1 || got[0] != "a" {
			t.Fatalf("got = %#v", got)
		}
	})

	t.Run("two distinct pages succeed", func(t *testing.T) {
		t.Parallel()
		var calls int
		got, err := collectAllPages("", func(pageToken string) ([]string, string, error) {
			calls++
			switch pageToken {
			case "":
				return []string{"a"}, "page-2", nil
			case "page-2":
				return []string{"b"}, "", nil
			default:
				t.Fatalf("unexpected pageToken %q", pageToken)
				return nil, "", nil
			}
		})
		if err != nil {
			t.Fatalf("err = %v", err)
		}
		if calls != 2 {
			t.Fatalf("calls = %d, want 2", calls)
		}
		if len(got) != 2 || got[0] != "a" || got[1] != "b" {
			t.Fatalf("got = %#v", got)
		}
	})

	t.Run("repeated next page token returns an error", func(t *testing.T) {
		t.Parallel()
		var calls int
		got, err := collectAllPages("", func(pageToken string) ([]string, string, error) {
			calls++
			if calls > 4 {
				return nil, "", errors.New("fetch called after repeated token")
			}
			return []string{"a"}, "stuck", nil
		})
		if err == nil || !strings.Contains(err.Error(), "repeated page token") {
			t.Fatalf("err = %v", err)
		}
		if got != nil {
			t.Fatalf("got = %#v, want nil", got)
		}
		if calls != 2 {
			t.Fatalf("calls = %d, want 2", calls)
		}
	})
}
