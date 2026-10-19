// Package library consulta o diário compartilhado e suas avaliações individuais.
package library

import (
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var ErrInvalid = errors.New("filtro ou paginação inválidos")

type Query struct {
	Person, Service, From, To string
	Pending                   bool
	GroupSeries               bool
	Limit                     int
	cursor                    *position
}
type position struct {
	Day string `json:"day"`
	ID  string `json:"id"`
}

func ValidID(id string) bool {
	if len(id) != 36 || id[8] != '-' || id[13] != '-' || id[18] != '-' || id[23] != '-' {
		return false
	}
	_, err := hex.DecodeString(strings.ReplaceAll(id, "-", ""))
	return err == nil
}
func ParseQuery(values url.Values) (Query, error) {
	q := Query{Person: values.Get("person"), Service: values.Get("service"), From: values.Get("from"), To: values.Get("to"), Limit: 50}
	allowed := map[string]bool{"person": true, "service": true, "from": true, "to": true, "pending": true, "group_series": true, "limit": true, "cursor": true}
	for key, v := range values {
		if !allowed[key] || len(v) != 1 {
			return q, ErrInvalid
		}
	}
	if q.Person != "" && !ValidID(q.Person) {
		return q, ErrInvalid
	}
	if q.Service != "" && !regexp.MustCompile(`^[a-z][a-z0-9_-]{0,39}$`).MatchString(q.Service) {
		return q, ErrInvalid
	}
	for _, day := range []string{q.From, q.To} {
		if day != "" {
			if _, err := time.Parse(time.DateOnly, day); err != nil {
				return q, ErrInvalid
			}
		}
	}
	if q.From != "" && q.To != "" && q.From > q.To {
		return q, ErrInvalid
	}
	if value := values.Get("pending"); value != "" {
		if value != "true" && value != "false" {
			return q, ErrInvalid
		}
		q.Pending = value == "true"
	}
	if value := values.Get("group_series"); value != "" {
		if value != "true" && value != "false" {
			return q, ErrInvalid
		}
		q.GroupSeries = value == "true"
	}
	if value := values.Get("limit"); value != "" {
		n, err := strconv.Atoi(value)
		if err != nil || n < 1 || n > 100 {
			return q, ErrInvalid
		}
		q.Limit = n
	}
	if value := values.Get("cursor"); value != "" {
		if len(value) > 500 {
			return q, ErrInvalid
		}
		raw, err := base64.RawURLEncoding.Strict().DecodeString(value)
		if err != nil {
			return q, ErrInvalid
		}
		var p position
		if json.Unmarshal(raw, &p) != nil || !ValidID(p.ID) {
			return q, ErrInvalid
		}
		if _, err = time.Parse(time.DateOnly, p.Day); err != nil {
			return q, ErrInvalid
		}
		q.cursor = &p
	}
	return q, nil
}
func cursorFor(day, id string) string {
	raw, _ := json.Marshal(position{Day: day, ID: id})
	return base64.RawURLEncoding.EncodeToString(raw)
}
