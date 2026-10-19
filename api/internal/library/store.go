package library

import (
	"context"
	"encoding/json"
	"fmt"
	"mywatchedlist/internal/catalogstore"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Store struct{ Pool *pgxpool.Pool }
type Person struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}
type Filters struct {
	People   []Person `json:"people"`
	Services []string `json:"services"`
}
type Connection struct {
	ID       string     `json:"id"`
	PersonID string     `json:"person_id"`
	Person   string     `json:"person"`
	Service  string     `json:"service"`
	Profile  string     `json:"profile"`
	Status   string     `json:"status"`
	LastSync *time.Time `json:"last_sync"`
}
type Rating struct {
	UserID    string    `json:"user_id"`
	Name      string    `json:"name"`
	Score     float64   `json:"score"`
	Comment   string    `json:"comment"`
	UpdatedAt time.Time `json:"updated_at"`
}
type RatingSummary struct {
	Ratings []Rating `json:"ratings"`
	Average *float64 `json:"average"`
	Count   int      `json:"ratings_count"`
}
type Item struct {
	ID               string     `json:"id"`
	PersonID         string     `json:"person_id"`
	Person           string     `json:"person"`
	Service          string     `json:"service"`
	Title            string     `json:"title"`
	SourceTitle      string     `json:"source_title"`
	MediaType        string     `json:"media_type"`
	WatchedOn        string     `json:"watched_on"`
	WatchedAt        *time.Time `json:"watched_at"`
	PosterPath       *string    `json:"poster_path"`
	PosterURL        string     `json:"poster_url"`
	MetadataProvider string     `json:"metadata_provider"`
	OverviewLanguage string     `json:"overview_language"`
	Year             *int       `json:"year"`
	Overview         string     `json:"overview"`
	Genres           []string   `json:"genres"`
	RatingKey        *string    `json:"rating_key"`
	People           []Person   `json:"people"`
	WatchCount       int        `json:"watch_count"`
	RatingSummary
}
type History struct {
	Items      []Item  `json:"items"`
	Total      int     `json:"total"`
	NextCursor *string `json:"next_cursor"`
}

func (s Store) Filters(ctx context.Context) (Filters, error) {
	result := Filters{People: []Person{}, Services: []string{}}
	rows, err := s.Pool.Query(ctx, `SELECT id::text,name FROM users ORDER BY lower(name),id`)
	if err != nil {
		return result, err
	}
	for rows.Next() {
		var p Person
		if err = rows.Scan(&p.ID, &p.Name); err != nil {
			rows.Close()
			return result, err
		}
		result.People = append(result.People, p)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return result, err
	}
	rows, err = s.Pool.Query(ctx, `SELECT DISTINCT service FROM streaming_accounts ORDER BY service`)
	if err != nil {
		return result, err
	}
	defer rows.Close()
	for rows.Next() {
		var service string
		if err = rows.Scan(&service); err != nil {
			return result, err
		}
		result.Services = append(result.Services, service)
	}
	return result, rows.Err()
}
func (s Store) Connections(ctx context.Context) ([]Connection, error) {
	result := []Connection{}
	rows, err := s.Pool.Query(ctx, `SELECT a.id::text,a.user_id::text,u.name,a.service,a.profile_name,a.session_status,a.last_sync_at FROM streaming_accounts a JOIN users u ON u.id=a.user_id ORDER BY a.service,lower(u.name),a.id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var c Connection
		if err = rows.Scan(&c.ID, &c.PersonID, &c.Person, &c.Service, &c.Profile, &c.Status, &c.LastSync); err != nil {
			return nil, err
		}
		result = append(result, c)
	}
	return result, rows.Err()
}

// As expressões são constantes internas, nunca entrada do cliente. Valores da
// consulta continuam sendo parâmetros pgx, inclusive chave/UUID nas mutações.
func ratingSelect(key, title string) string {
	return fmt.Sprintf(`
 SELECT coalesce(jsonb_agg(jsonb_build_object('user_id',v.user_id::text,'name',u.name,'score',v.score::float8,'comment',coalesce(v.comment,''),'updated_at',v.updated_at) ORDER BY u.name),'[]'::jsonb),avg(v.score)::float8,count(*)::int
 FROM (SELECT DISTINCT ON(r.user_id) r.user_id,r.score,r.comment,r.updated_at FROM ratings r
 WHERE r.source_key=%s OR r.title_id=%s
 ORDER BY r.user_id,(r.source_key=%s) DESC NULLS LAST,r.updated_at DESC) v JOIN users u ON u.id=v.user_id`, key, title, key)
}

const historyBase = `WITH base AS (
 SELECT e.*,a.service,u.name AS person,coalesce(nullif(o.display_title,''),t.title,CASE WHEN e.media_type_hint='tv' AND e.match_title<>'' THEN e.match_title ELSE e.source_title END) AS display_title,
 coalesce(o.media_type,t.media_type,nullif(e.media_type_hint,''),'unknown') AS display_type,t.poster_path,t.release_year,coalesce(nullif(o.overview,''),t.overview,'') AS overview,coalesce(t.genres,'{}'::text[]) AS genres,
 coalesce(o.poster_url,'') AS poster_url,coalesce(o.provider,CASE WHEN t.id IS NOT NULL THEN 'tmdb' ELSE '' END) AS metadata_provider,CASE WHEN o.overview<>'' THEN o.overview_language ELSE coalesce(t.overview_language,'') END AS overview_language,
 CASE WHEN e.source_work_id<>'' THEN a.service||':work:'||e.source_work_id
 WHEN e.media_type_hint<>'tv' AND nullif(e.source_title_id,'') IS NOT NULL THEN a.service||':work:'||e.source_title_id ELSE NULL END AS rating_key
 FROM watch_events e JOIN streaming_accounts a ON a.id=e.streaming_account_id JOIN users u ON u.id=e.user_id LEFT JOIN metadata_overrides o ON o.source_key=(` + catalogstore.SourceKey + `) LEFT JOIN titles t ON t.id=coalesce(o.title_id,e.title_id)
 WHERE ($1::uuid IS NULL OR e.user_id=$1::uuid) AND ($2::text IS NULL OR a.service=$2) AND ($3::date IS NULL OR e.watched_on >= $3::date) AND ($4::date IS NULL OR e.watched_on <= $4::date)
), identified AS (
 SELECT b.*,CASE WHEN $7::bool AND b.display_type='tv' AND b.rating_key IS NOT NULL THEN b.rating_key ELSE b.id::text END AS list_key FROM base b
), personal AS (
 SELECT b.*,EXISTS(SELECT 1 FROM ratings r WHERE r.user_id=$5::uuid AND (r.source_key=b.rating_key OR r.title_id=b.title_id)) AS is_rated,
 row_number() OVER(PARTITION BY coalesce(b.rating_key,b.id::text) ORDER BY b.watched_on DESC,b.id DESC) AS work_rank,
 row_number() OVER(PARTITION BY b.list_key ORDER BY b.watched_on DESC,b.id DESC) AS list_rank FROM identified b
), filtered AS (
 SELECT * FROM personal WHERE list_rank=1 AND (NOT $6::bool OR (rating_key IS NOT NULL AND NOT is_rated AND work_rank=1))
), summaries AS (
 SELECT list_key,sum(events)::int AS watch_count,jsonb_agg(jsonb_build_object('id',user_id::text,'name',person) ORDER BY lower(person),user_id) AS people
 FROM (SELECT list_key,user_id,person,count(*) AS events FROM identified GROUP BY list_key,user_id,person) members GROUP BY list_key
) `

func nullable(value string) any {
	if value == "" {
		return nil
	}
	return value
}
func dateValue(value string) any {
	if value == "" {
		return nil
	}
	day, _ := time.Parse(time.DateOnly, value)
	return day
}

func (s Store) History(ctx context.Context, actor string, q Query) (History, error) {
	result := History{Items: []Item{}}
	args := []any{nullable(q.Person), nullable(q.Service), dateValue(q.From), dateValue(q.To), actor, q.Pending, q.GroupSeries}
	tx, err := s.Pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return result, err
	}
	defer tx.Rollback(context.Background())
	if err = tx.QueryRow(ctx, historyBase+`SELECT count(*)::int FROM filtered`, args...).Scan(&result.Total); err != nil {
		return result, err
	}
	var cursorDay, cursorID any
	if q.cursor != nil {
		cursorDay = dateValue(q.cursor.Day)
		cursorID = q.cursor.ID
	}
	args = append(args, cursorDay, cursorID, q.Limit+1)
	query := historyBase + `SELECT f.id::text,f.user_id::text,f.person,f.service,f.display_title,f.source_title,f.display_type,f.watched_on::text,f.watched_at,f.poster_path,f.release_year,f.overview,f.genres,f.rating_key,g.people,g.watch_count,f.poster_url,f.metadata_provider,f.overview_language,a.*
 FROM filtered f JOIN summaries g ON g.list_key=f.list_key LEFT JOIN LATERAL (` + ratingSelect("f.rating_key", "f.title_id") + `) a ON true
 WHERE ($8::date IS NULL OR (f.watched_on,f.id)<($8::date,$9::uuid)) ORDER BY f.watched_on DESC,f.id DESC LIMIT $10`
	rows, err := tx.Query(ctx, query, args...)
	if err != nil {
		return result, err
	}
	for rows.Next() {
		var item Item
		var ratings, people []byte
		if err = rows.Scan(&item.ID, &item.PersonID, &item.Person, &item.Service, &item.Title, &item.SourceTitle, &item.MediaType, &item.WatchedOn, &item.WatchedAt, &item.PosterPath, &item.Year, &item.Overview, &item.Genres, &item.RatingKey, &people, &item.WatchCount, &item.PosterURL, &item.MetadataProvider, &item.OverviewLanguage, &ratings, &item.Average, &item.Count); err != nil {
			rows.Close()
			return result, err
		}
		if err = json.Unmarshal(ratings, &item.Ratings); err != nil {
			rows.Close()
			return result, err
		}
		if err = json.Unmarshal(people, &item.People); err != nil {
			rows.Close()
			return result, err
		}
		result.Items = append(result.Items, item)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return result, err
	}
	if len(result.Items) > q.Limit {
		result.Items = result.Items[:q.Limit]
		last := result.Items[len(result.Items)-1]
		cursor := cursorFor(last.WatchedOn, last.ID)
		result.NextCursor = &cursor
	}
	if err = tx.Commit(ctx); err != nil {
		return result, err
	}
	return result, nil
}
