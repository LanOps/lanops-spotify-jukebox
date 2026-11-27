package jukebox

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestPlaylistItemEndpoints(t *testing.T) {
	type call struct{ method, path, query, body string }
	var calls []call
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		calls = append(calls, call{r.Method, r.URL.Path, r.URL.RawQuery, string(b)})
		if r.Method != http.MethodGet {
			_, _ = w.Write([]byte(`{"snapshot_id":"x"}`))
			return
		}
		if r.URL.Query().Get("offset") == "" {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"items": []any{
					map[string]any{"item": map[string]any{"id": "new1", "name": "New Field"}},
					map[string]any{"item": nil},
				},
				"next": srv.URL + "/playlists/pl/items?offset=50",
			})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"items": []any{map[string]any{"track": map[string]any{"id": "old1", "name": "Old Field"}}},
		})
	}))
	defer srv.Close()

	old := spotifyApiBaseUrl
	spotifyApiBaseUrl = srv.URL + "/"
	defer func() { spotifyApiBaseUrl = old }()

	c := &Client{}
	c.spotify.httpClient = srv.Client()
	ctx := context.Background()

	tracks, err := c.getPlaylistTracks(ctx, "pl")
	if err != nil {
		t.Fatal(err)
	}
	if len(tracks) != 2 || tracks[0].ID != "new1" || tracks[1].ID != "old1" {
		t.Fatalf("unexpected tracks: %+v", tracks)
	}
	if err := c.removeTrackFromPlaylist(ctx, "pl", "abc"); err != nil {
		t.Fatal(err)
	}
	if err := c.addTrackToPlaylist(ctx, "pl", "abc"); err != nil {
		t.Fatal(err)
	}

	want := []call{
		{"GET", "/playlists/pl/items", "limit=50&additional_types=track", ""},
		{"GET", "/playlists/pl/items", "offset=50", ""},
		{"DELETE", "/playlists/pl/items", "", `{"items":[{"uri":"spotify:track:abc"}]}`},
		{"POST", "/playlists/pl/items", "", `{"uris":["spotify:track:abc"]}`},
	}
	if len(calls) != len(want) {
		t.Fatalf("got %d calls, want %d: %+v", len(calls), len(want), calls)
	}
	for i := range want {
		if calls[i] != want[i] {
			t.Errorf("call %d: got %+v, want %+v", i, calls[i], want[i])
		}
	}
}

func TestPlaylistRequestWithoutLogin(t *testing.T) {
	c := &Client{}
	if _, err := c.getPlaylistTracks(context.Background(), "pl"); err != errNoSpotifyClient {
		t.Fatalf("expected errNoSpotifyClient, got %v", err)
	}
}
