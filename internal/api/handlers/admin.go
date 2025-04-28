package handlers

import (
	"fmt"
	"net/http"

	"github.com/PedroMartini98/Twitter-Clone.go.git/internal/api/middleware"
	"github.com/PedroMartini98/Twitter-Clone.go.git/internal/database"
	"github.com/PedroMartini98/Twitter-Clone.go.git/internal/utils"
)

type AdminHandler struct {
	dbQueries *database.Queries
	platform  string
	metrics   *middleware.MetricsMiddleware
}

func NewAdminHandler(dbQueries *database.Queries, platform string, metrics *middleware.MetricsMiddleware) *AdminHandler {
	return &AdminHandler{
		dbQueries: dbQueries,
		platform:  platform,
		metrics:   metrics,
	}
}

func (h *AdminHandler) GetMetricts(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)

	htmlTemplate := `
<html>
  <body>
    <h1>Welcome, Chirpy Admin</h1>
    <p>Chirpy has been visited %d times!</p>
  </body>
</html>`

	fmt.Fprintf(w, htmlTemplate, h.metrics.GetHits())

}

func (h *AdminHandler) ResetServer(w http.ResponseWriter, r *http.Request) {

	if h.platform != "dev" {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	err := h.dbQueries.DeleteAllUsers(r.Context())
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Failed to delete all users")
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Users deleted"))
}
