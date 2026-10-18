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
	payload, _, err := activityData(body)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(payload, &envelope); err != nil || len(envelope.ViewedItems) == 0 || bytes.Equal(bytes.TrimSpace(envelope.ViewedItems), []byte("null")) {
		return nil, collector.ErrParserChanged
	}
	var items []struct {
		MovieID     json.RawMessage `json:"movieID"`
		Title       *string         `json:"title"`
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
		if err != nil || dateErr != nil || item.Title == nil || stamp < 852076800000 || stamp >= 4102444800000 {
			return nil, collector.ErrParserChanged
		}
		title := strings.TrimSpace(*item.Title)
		label := title
		if label == "" {
			label = "Título indisponível na Netflix"
		}
		instant := time.UnixMilli(stamp).UTC()
		key := fmt.Sprintf("netflix:%s:%d", id, stamp)
		e := collector.Event{SourceKey: key, SourceTitleID: id, SourceWorkID: id, SourceTitle: label, WatchedOn: instant.In(zone).Format(time.DateOnly), WatchedAt: &instant, MatchTitle: title}
		if len(item.Series) > 0 && string(item.Series) != "null" && string(item.Series) != "0" {
			workID, err := positiveID(item.Series)
			if err != nil {
				return nil, collector.ErrParserChanged
			}
			e.SourceWorkID = workID
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
		if title == "" {
			e.MatchTitle = ""
		}
		if !seen[key] {
			events = append(events, e)
			seen[key] = true
		}
	}
	return events, nil
}

// activityData reconhece tanto vaModel.data (primeira página) quanto o atom
// Falcor observado na paginação atual. Não procura arrays arbitrários no JSON.
func activityData(body []byte) ([]byte, string, error) {
	var root map[string]json.RawMessage
	if json.Unmarshal(body, &root) != nil {
		return nil, "", collector.ErrParserChanged
	}
	payload := body
	if _, ok := root["viewedItems"]; !ok {
		var graph struct {
			JSONGraph struct {
				AUI struct {
					Activity struct {
						Value json.RawMessage `json:"value"`
					} `json:"viewingActivity"`
				} `json:"aui"`
			} `json:"jsonGraph"`
		}
		if json.Unmarshal(body, &graph) != nil || len(graph.JSONGraph.AUI.Activity.Value) == 0 {
			return nil, "", collector.ErrParserChanged
		}
		payload = graph.JSONGraph.AUI.Activity.Value
	}
	var info struct {
		Profile struct {
			GUID string `json:"guid"`
		} `json:"profileInfo"`
	}
	if json.Unmarshal(payload, &info) != nil {
		return nil, "", collector.ErrParserChanged
	}
	return payload, info.Profile.GUID, nil
}

func positiveID(raw json.RawMessage) (string, error) {
	s := strings.Trim(string(raw), `"`)
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil || n <= 0 {
		return "", collector.ErrParserChanged
	}
	return strconv.FormatInt(n, 10), nil
}
