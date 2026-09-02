package cmd

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestFetchBackupChatSpacesRejectsRepeatedPageToken(t *testing.T) {
	var calls atomic.Int32
	svc := newChatTestService(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !(r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/spaces")) {
			http.NotFound(w, r)
			return
		}
		if calls.Add(1) > 2 {
			http.Error(w, "unexpected extra space page request", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"spaces":        []map[string]any{{"name": "spaces/aaa"}},
			"nextPageToken": "stuck",
		})
	}))

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_, err := fetchBackupChatSpaces(ctx, svc)
	if err == nil || !strings.Contains(err.Error(), "repeated page token") {
		t.Fatalf("err = %v after %d list calls", err, calls.Load())
	}
	if got := calls.Load(); got != 2 {
		t.Fatalf("list calls = %d, want 2", got)
	}
}

func TestFetchBackupClassroomCoursesRejectsRepeatedPageToken(t *testing.T) {
	var calls atomic.Int32
	svc, closeService := newClassroomTestService(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !(r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/courses")) {
			http.NotFound(w, r)
			return
		}
		if calls.Add(1) > 2 {
			http.Error(w, "unexpected extra course page request", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"courses":       []map[string]any{{"id": "c1", "name": "Biology"}},
			"nextPageToken": "stuck",
		})
	}))
	defer closeService()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_, err := fetchBackupClassroomCourses(ctx, svc)
	if err == nil || !strings.Contains(err.Error(), "repeated page token") {
		t.Fatalf("err = %v after %d list calls", err, calls.Load())
	}
	if got := calls.Load(); got != 2 {
		t.Fatalf("list calls = %d, want 2", got)
	}
}

func TestFetchBackupChatSpacesTwoDistinctPagesSucceed(t *testing.T) {
	var calls atomic.Int32
	svc := newChatTestService(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !(r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/spaces")) {
			http.NotFound(w, r)
			return
		}
		n := calls.Add(1)
		w.Header().Set("Content-Type", "application/json")
		if n == 1 {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"spaces":        []map[string]any{{"name": "spaces/aaa"}},
				"nextPageToken": "page-2",
			})
			return
		}
		if n == 2 {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"spaces": []map[string]any{{"name": "spaces/bbb"}},
			})
			return
		}
		http.Error(w, "unexpected extra space page request", http.StatusBadRequest)
	}))

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	got, err := fetchBackupChatSpaces(ctx, svc)
	if err != nil {
		t.Fatalf("err = %v after %d list calls", err, calls.Load())
	}
	if calls.Load() != 2 {
		t.Fatalf("list calls = %d, want 2", calls.Load())
	}
	if len(got) != 2 || got[0].Name != "spaces/aaa" || got[1].Name != "spaces/bbb" {
		t.Fatalf("spaces = %#v", got)
	}
}
