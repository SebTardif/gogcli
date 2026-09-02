package cmd

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"google.golang.org/api/chat/v1"
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

func TestFetchBackupChatSpacesEmptyNextTokenEnds(t *testing.T) {
	var calls atomic.Int32
	svc := newChatTestService(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !(r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/spaces")) {
			http.NotFound(w, r)
			return
		}
		calls.Add(1)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"spaces": []map[string]any{{"name": "spaces/aaa"}},
		})
	}))

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	got, err := fetchBackupChatSpaces(ctx, svc)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if calls.Load() != 1 {
		t.Fatalf("list calls = %d, want 1", calls.Load())
	}
	if len(got) != 1 || got[0].Name != "spaces/aaa" {
		t.Fatalf("spaces = %#v", got)
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
			if r.URL.Query().Get("pageToken") != "page-2" {
				t.Errorf("pageToken = %q, want page-2", r.URL.Query().Get("pageToken"))
			}
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

func TestFetchBackupChatMessagesRejectsRepeatedPageToken(t *testing.T) {
	var calls atomic.Int32
	svc := newChatTestService(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !(r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/messages")) {
			http.NotFound(w, r)
			return
		}
		if calls.Add(1) > 2 {
			http.Error(w, "unexpected extra message page request", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"messages":      []map[string]any{{"name": "spaces/aaa/messages/m1"}},
			"nextPageToken": "stuck",
		})
	}))

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_, err := fetchBackupChatMessages(ctx, svc, []*chat.Space{{Name: "spaces/aaa"}})
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

func TestFetchBackupClassroomChildrenRejectRepeatedPageTokens(t *testing.T) {
	tests := []struct {
		name        string
		path        string
		responseKey string
	}{
		{name: "topics", path: "/topics", responseKey: "topic"},
		{name: "announcements", path: "/announcements", responseKey: "announcements"},
		{name: "coursework", path: "/courseWork", responseKey: "courseWork"},
		{name: "materials", path: "/courseWorkMaterials", responseKey: "courseWorkMaterial"},
		{name: "submissions", path: "/studentSubmissions", responseKey: "studentSubmissions"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var calls atomic.Int32
			svc, closeService := newClassroomTestService(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if !(r.Method == http.MethodGet && strings.Contains(r.URL.Path, tc.path)) {
					http.NotFound(w, r)
					return
				}
				if calls.Add(1) > 2 {
					http.Error(w, "unexpected extra page request", http.StatusBadRequest)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(map[string]any{
					tc.responseKey:  []map[string]any{{"id": "item-1"}},
					"nextPageToken": "stuck",
				})
			}))
			defer closeService()

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			var err error
			switch tc.name {
			case "topics":
				_, err = fetchClassroomTopicsBestEffort(ctx, svc, "c1")
			case "announcements":
				_, err = fetchClassroomAnnouncementsBestEffort(ctx, svc, "c1")
			case "coursework":
				_, err = fetchClassroomCourseWorkBestEffort(ctx, svc, "c1")
			case "materials":
				_, err = fetchClassroomMaterialsBestEffort(ctx, svc, "c1")
			case "submissions":
				_, err = fetchClassroomSubmissionsBestEffort(ctx, svc, "c1")
			}
			if err == nil || !strings.Contains(err.Error(), "repeated page token") {
				t.Fatalf("err = %v after %d list calls", err, calls.Load())
			}
			if got := calls.Load(); got != 2 {
				t.Fatalf("list calls = %d, want 2", got)
			}
		})
	}
}

func TestFetchBackupClassroomCoursesTwoDistinctPagesSucceed(t *testing.T) {
	var calls atomic.Int32
	svc, closeService := newClassroomTestService(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !(r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/courses")) {
			http.NotFound(w, r)
			return
		}
		n := calls.Add(1)
		w.Header().Set("Content-Type", "application/json")
		if n == 1 {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"courses":       []map[string]any{{"id": "c1", "name": "Biology"}},
				"nextPageToken": "page-2",
			})
			return
		}
		if n == 2 {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"courses": []map[string]any{{"id": "c2", "name": "Chemistry"}},
			})
			return
		}
		http.Error(w, "unexpected extra course page request", http.StatusBadRequest)
	}))
	defer closeService()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	got, err := fetchBackupClassroomCourses(ctx, svc)
	if err != nil {
		t.Fatalf("err = %v after %d list calls", err, calls.Load())
	}
	if calls.Load() != 2 {
		t.Fatalf("list calls = %d, want 2", calls.Load())
	}
	if len(got) != 2 || got[0].Id != "c1" || got[1].Id != "c2" {
		t.Fatalf("courses = %#v", got)
	}
}
