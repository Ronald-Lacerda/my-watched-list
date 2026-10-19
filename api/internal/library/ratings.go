package library

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
)

var (
	ErrNotFound     = errors.New("registro não encontrado")
	ErrSyncRequired = errors.New("sincronize esta conexão antes de avaliar a série")
)

func ValidateRating(score float64, comment string) error {
	if math.IsNaN(score) || math.IsInf(score, 0) || score < 0 || score > 10 || math.Mod(score, 0.5) != 0 || !utf8.ValidString(comment) || utf8.RuneCountInString(comment) > 2000 {
		return ErrInvalid
	}
	return nil
}
func (s Store) SaveRating(ctx context.Context, actor, eventID string, score float64, comment string) (RatingSummary, error) {
	if err := ValidateRating(score, comment); err != nil {
		return RatingSummary{}, err
	}
	return s.mutateRating(ctx, actor, eventID, &score, comment)
}
func (s Store) DeleteRating(ctx context.Context, actor, eventID string) (RatingSummary, error) {
	return s.mutateRating(ctx, actor, eventID, nil, "")
}

func (s Store) mutateRating(ctx context.Context, actor, eventID string, score *float64, comment string) (RatingSummary, error) {
	result := RatingSummary{Ratings: []Rating{}}
	if !ValidID(eventID) {
		return result, ErrInvalid
	}
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return result, err
	}
	defer tx.Rollback(context.Background())
	var key, titleID *string
	err = tx.QueryRow(ctx, `SELECT CASE WHEN e.source_work_id<>'' THEN a.service||':work:'||e.source_work_id WHEN e.media_type_hint<>'tv' AND nullif(e.source_title_id,'') IS NOT NULL THEN a.service||':work:'||e.source_title_id ELSE NULL END,e.title_id::text FROM watch_events e JOIN streaming_accounts a ON a.id=e.streaming_account_id WHERE e.id=$1 FOR SHARE OF e,a`, eventID).Scan(&key, &titleID)
	if errors.Is(err, pgx.ErrNoRows) {
		return result, ErrNotFound
	}
	if err != nil {
		return result, err
	}
	if key == nil {
		return result, ErrSyncRequired
	}
	if score != nil {
		_, err = tx.Exec(ctx, `INSERT INTO ratings(user_id,source_key,score,comment) VALUES($1,$2,$3,$4) ON CONFLICT(user_id,source_key) DO UPDATE SET score=excluded.score,comment=excluded.comment,updated_at=now()`, actor, *key, *score, comment)
		if err != nil {
			return result, err
		}
		// A edição passa a usar a identidade própria e não deixa uma nota antiga
		// do mesmo usuário reaparecer como fallback após excluir.
		_, err = tx.Exec(ctx, `DELETE FROM ratings WHERE user_id=$1 AND source_key IS NULL AND title_id=$2`, actor, titleID)
	} else {
		_, err = tx.Exec(ctx, `DELETE FROM ratings WHERE user_id=$1 AND (source_key=$2 OR (source_key IS NULL AND title_id=$3))`, actor, *key, titleID)
	}
	if err != nil {
		return result, err
	}
	var raw []byte
	if err = tx.QueryRow(ctx, ratingSelect("$1::text", "$2::uuid"), *key, titleID).Scan(&raw, &result.Average, &result.Count); err != nil {
		return result, err
	}
	if err = json.Unmarshal(raw, &result.Ratings); err != nil {
		return result, err
	}
	if err = tx.Commit(ctx); err != nil {
		return result, err
	}
	return result, nil
}
