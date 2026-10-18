package httpapi

import (
	"mywatchedlist/api/internal/library"
	"net/http"
)

func (a *app) filters(w http.ResponseWriter, r *http.Request) {
	data, err := (library.Store{Pool: a.pool}).Filters(r.Context())
	if err != nil {
		a.authError(w, err)
		return
	}
	reply(w, 200, data)
}
func (a *app) history(w http.ResponseWriter, r *http.Request) {
	q, err := library.ParseQuery(r.URL.Query())
	if err != nil {
		apiError(w, 400, "invalid_filters", "Revise pessoa, serviço, datas e paginação.")
		return
	}
	data, err := (library.Store{Pool: a.pool}).History(r.Context(), currentUser(r).ID, q)
	if err != nil {
		a.authError(w, err)
		return
	}
	reply(w, 200, data)
}
func (a *app) connections(w http.ResponseWriter, r *http.Request) {
	data, err := (library.Store{Pool: a.pool}).Connections(r.Context())
	if err != nil {
		a.authError(w, err)
		return
	}
	reply(w, 200, map[string]any{"connections": data})
}
