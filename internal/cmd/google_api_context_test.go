package cmd

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	youtube "google.golang.org/api/youtube/v3"

	"github.com/openclaw/gogcli/internal/app"
)

func TestYouTubePlaylistsListHonorsCanceledContext(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	svc := newGoogleTestServiceWithEndpoint(t, srv.Client(), srv.URL+"/", youtube.NewService)
	ctx := withYouTubeTestServices(newCmdRuntimeOutputContext(t, io.Discard, io.Discard), youtubeTestServices{
		Account: fixedYouTubeTestService(svc),
	})
	ctx, cancel := context.WithCancel(ctx)
	cancel()

	err := runKong(t, &YouTubePlaylistsListCmd{}, []string{"--mine"}, ctx, &RootFlags{
		Account: "me@example.com",
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want context.Canceled", err)
	}
}

func TestChatMessagesListUnreadHonorsCanceledContext(t *testing.T) {
	svc := useFakeChatService(t, func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "spaceReadState") {
			http.NotFound(w, r)
			return
		}
		<-r.Context().Done()
		w.WriteHeader(http.StatusOK)
	})
	ctx := withTestRuntime(newCmdRuntimeOutputContext(t, io.Discard, io.Discard), func(runtime *app.Runtime) {
		runtime.Services.Chat = chatTestServices.fixed(svc)
	})
	ctx, cancel := context.WithCancel(ctx)
	cancel()

	err := runKong(t, &ChatMessagesListCmd{}, []string{"spaces/aaa", "--unread"}, ctx, &RootFlags{
		Account: "a@b.com",
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want context.Canceled", err)
	}
}

func TestPeopleProfileGetHonorsCanceledContext(t *testing.T) {
	svc, closeSrv := newPeopleService(t, func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
		w.WriteHeader(http.StatusOK)
	})
	t.Cleanup(closeSrv)

	ctx := withPeopleTestServices(newCmdRuntimeOutputContext(t, io.Discard, io.Discard), peopleTestServices{
		Directory: fixedPeopleTestService(svc),
	})
	ctx, cancel := context.WithCancel(ctx)
	cancel()

	err := runKong(t, &PeopleGetCmd{}, []string{"people/123"}, ctx, &RootFlags{
		Account: "a@b.com",
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want context.Canceled", err)
	}
}
