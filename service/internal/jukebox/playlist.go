package jukebox

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/zmb3/spotify/v2"
)

// Spotify's February 2026 Web API changes removed the /playlists/{id}/tracks
// endpoints in favour of /playlists/{id}/items, and renamed the per-item
// "track" field to "item". zmb3/spotify v2.4.3 still calls the old paths, so
// the jukebox talks to the playlist item endpoints directly.
// See https://developer.spotify.com/documentation/web-api/tutorials/february-2026-migration-guide

var spotifyApiBaseUrl = "https://api.spotify.com/v1/"

var errNoSpotifyClient = errors.New("spotify client not logged in")

type playlistItemsPage struct {
	Items []playlistItem `json:"items"`
	Next  string         `json:"next"`
}

type playlistItem struct {
	Item  *spotify.FullTrack `json:"item"`
	Track *spotify.FullTrack `json:"track"` // pre-2026 field name, kept as a fallback
}

func (p playlistItem) fullTrack() *spotify.FullTrack {
	if p.Item != nil {
		return p.Item
	}
	return p.Track
}

func (c *Client) spotifyRequest(ctx context.Context, method, path string, body any, out any) error {
	if c.spotify.httpClient == nil {
		return errNoSpotifyClient
	}
	var reqBody io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reqBody = bytes.NewReader(b)
	}
	// path is either relative to the API base, or a "next" URL returned by Spotify.
	// Only ever send the bearer token to the Spotify API itself.
	url := spotifyApiBaseUrl + path
	if strings.Contains(path, "://") {
		if !strings.HasPrefix(path, spotifyApiBaseUrl) {
			return fmt.Errorf("refusing to call non-Spotify URL %q", path)
		}
		url = path
	}
	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.spotify.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		msg, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return fmt.Errorf("spotify %s %s: %s: %s", method, path, resp.Status, bytes.TrimSpace(msg))
	}
	if out != nil {
		return json.NewDecoder(resp.Body).Decode(out)
	}
	return nil
}

// getPlaylistTracks returns every track in a playlist, following pagination.
func (c *Client) getPlaylistTracks(ctx context.Context, playlistId spotify.ID) ([]*spotify.FullTrack, error) {
	var tracks []*spotify.FullTrack
	next := fmt.Sprintf("playlists/%s/items?limit=50&additional_types=track", playlistId)
	for next != "" {
		var page playlistItemsPage
		if err := c.spotifyRequest(ctx, http.MethodGet, next, nil, &page); err != nil {
			return nil, err
		}
		for _, item := range page.Items {
			if t := item.fullTrack(); t != nil && t.ID != "" {
				tracks = append(tracks, t)
			}
		}
		next = page.Next
	}
	return tracks, nil
}

func (c *Client) addTrackToPlaylist(ctx context.Context, playlistId, trackId spotify.ID) error {
	body := map[string][]string{"uris": {trackUri(trackId)}}
	return c.spotifyRequest(ctx, http.MethodPost, fmt.Sprintf("playlists/%s/items", playlistId), body, nil)
}

func (c *Client) removeTrackFromPlaylist(ctx context.Context, playlistId, trackId spotify.ID) error {
	body := map[string][]map[string]string{"items": {{"uri": trackUri(trackId)}}}
	return c.spotifyRequest(ctx, http.MethodDelete, fmt.Sprintf("playlists/%s/items", playlistId), body, nil)
}

func trackUri(id spotify.ID) string {
	return fmt.Sprintf("spotify:track:%s", id)
}
