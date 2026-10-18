package httpapi

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"mywatchedlist/api/internal/library"
)

func (a *app) rating(w http.ResponseWriter, r *http.Request) {
	if !a.origin(w, r) {
		return
	}
	if r.URL.RawQuery != "" {
		apiError(w, 400, "invalid_rating", "Informe apenas sua nota e comentário.")
		return
	}
	id := chi.URLParam(r, "id")
	store := library.Store{Pool: a.pool}
	var result library.RatingSummary
	var err error
	if r.Method == http.MethodPut {
		var body struct {
			Score   *float64 `json:"score"`
			Comment string   `json:"comment"`
		}
		if !readJSON(w, r, &body) {
			return
		}
		if body.Score == nil {
			apiError(w, 400, "invalid_rating", "Escolha uma nota de 0 a 10, em passos de 0,5.")
			return
		}
		result, err = store.SaveRating(r.Context(), currentUser(r).ID, id, *body.Score, body.Comment)
	} else {
		if r.ContentLength != 0 {
			var empty struct{}
			if !readJSON(w, r, &empty) {
				return
			}
		}
		result, err = store.DeleteRating(r.Context(), currentUser(r).ID, id)
	}
	if err != nil {
		switch {
		case errors.Is(err, library.ErrInvalid):
			apiError(w, 400, "invalid_rating", "Use uma nota de 0 a 10 em passos de 0,5 e comentário de até 2000 caracteres.")
		case errors.Is(err, library.ErrNotFound):
			apiError(w, 404, "not_found", "Não encontramos este registro.")
		case errors.Is(err, library.ErrSyncRequired):
			apiError(w, 409, "sync_required", "Esta série precisa de uma nova sincronização antes de receber nota.")
		default:
			a.authError(w, err)
		}
		return
	}
	reply(w, 200, result)
}
