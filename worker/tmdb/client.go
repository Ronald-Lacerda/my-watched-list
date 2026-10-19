// Package tmdb adapta o cliente compartilhado ao contrato do coletor.
package tmdb

import (
	"context"
	"mywatchedlist/internal/metadata"
	"mywatchedlist/worker/collector"
	"mywatchedlist/worker/store"
)

var (
	ErrUnavailable = metadata.ErrUnavailable
	ErrNotFound    = metadata.ErrNotFound
	ErrAmbiguous   = metadata.ErrAmbiguous
)

type Client struct{ client *metadata.Client }

func New(token string) *Client { return &Client{client: metadata.New(token)} }

func (c *Client) Match(ctx context.Context, e collector.Event) (store.Title, error) {
	return c.client.Match(ctx, metadata.MatchInput{MatchTitle: e.MatchTitle, MediaType: e.MediaType, SeasonNumber: e.SeasonNumber, EpisodeNumber: e.EpisodeNumber})
}
