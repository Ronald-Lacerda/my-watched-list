// Package netflix lê respostas da atividade de visualização, sem parsear linhas HTML.
package netflix

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"mywatchedlist/worker/collector"
)

// Normalize reconhece o envelope viewedItems. Campo ausente/null é uma quebra
// de contrato; array vazio explícito é histórico vazio válido.
func Normalize(body []byte, zone *time.Location) ([]collector.Event, error) {
	if zone == nil {
		return nil, collector.ErrParserChanged
	}
	var envelope struct {
		ViewedItems json.RawMessage `json:"viewedItems"`
	}
	dec := json.NewDecoder(bytes.NewReader(body))
	if err := dec.Decode(&envelope); err != nil || len(envelope.ViewedItems) == 0 || bytes.Equal(bytes.TrimSpace(envelope.ViewedItems), []byte("null")) {
		return nil, collector.ErrParserChanged
	}
	var items []struct {
		MovieID     json.RawMessage `json:"movieID"`
		Title       string          `json:"title"`
		Date        json.Number     `json:"date"`
		Series      json.RawMessage `json:"series"`
		SeriesTitle string          `json:"seriesTitle"`
		Season      *int            `json:"seasonNumber"`
		Episode     *int            `json:"episodeNumber"`
		VideoType   string          `json:"videoType"`
	}
	if err := json.Unmarshal(envelope.ViewedItems, &items); err != nil {
		return nil, collector.ErrParserChanged
	}
	events := make([]collector.Event, 0, len(items))
	seen := map[string]bool{}
	for _, item := range items {
		id, err := positiveID(item.MovieID)
		stamp, dateErr := item.Date.Int64()
		title := strings.TrimSpace(item.Title)
		if err != nil || dateErr != nil || title == "" || stamp < 852076800000 || stamp >= 4102444800000 {
			return nil, collector.ErrParserChanged
		}
		instant := time.UnixMilli(stamp).UTC()
		key := fmt.Sprintf("netflix:%s:%d", id, stamp)
		e := collector.Event{SourceKey: key, SourceTitleID: id, SourceTitle: title, WatchedOn: instant.In(zone).Format(time.DateOnly), WatchedAt: &instant, MatchTitle: title}
		if len(item.Series) > 0 && string(item.Series) != "null" && string(item.Series) != "0" {
			if _, err := positiveID(item.Series); err != nil {
				return nil, collector.ErrParserChanged
			}
			e.MediaType = "tv"
			e.MatchTitle = strings.TrimSpace(item.SeriesTitle)
		} else if item.VideoType == "movie" {
			e.MediaType = "movie"
		}
		if item.Season != nil || item.Episode != nil {
			if item.Season == nil || item.Episode == nil || *item.Season < 0 || *item.Episode < 1 || e.MediaType != "tv" {
				return nil, collector.ErrParserChanged
			}
			e.SeasonNumber = item.Season
			e.EpisodeNumber = item.Episode
		}
		if !seen[key] {
			events = append(events, e)
			seen[key] = true
		}
	}
	return events, nil
}

func positiveID(raw json.RawMessage) (string, error) {
	s := strings.Trim(string(raw), `"`)
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil || n <= 0 {
		return "", collector.ErrParserChanged
	}
	return strconv.FormatInt(n, 10), nil
}
