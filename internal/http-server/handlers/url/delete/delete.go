package delete

import (
	resp "URL-Shortener/internal/lib/api/response"
	"URL-Shortener/internal/lib/logger/sl"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"log/slog"
	"net/http"
)

type URLDelete interface {
	DeleteUrl(alias string) error
}

type Response struct {
	resp.Response
}

func New(log *slog.Logger, urlDelete URLDelete) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.url.delete"
		log = log.With(slog.String("operation", op))

		alias := chi.URLParam(r, "alias")
		if alias == "" {
			log.Error("missing alias")

			render.JSON(w, r, resp.Error("missing alias"))

			return

		}

		err := urlDelete.DeleteUrl(alias)
		if err != nil {
			log.Error("failed to delete url", sl.Err(err))

			render.JSON(w, r, resp.Error(err.Error()))

			return
		}

		log.Info("url deleted")
		responseOk(w, r)
	}

}

func responseOk(w http.ResponseWriter, r *http.Request) {
	render.JSON(w, r, Response{
		Response: resp.Ok(),
	})
}
