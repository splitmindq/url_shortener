package get

import (
	resp "URL-Shortener/internal/lib/api/response"
	"URL-Shortener/internal/lib/logger/sl"
	"URL-Shortener/internal/storage"
	"errors"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"log/slog"
	"net/http"
)

type URLGet interface {
	GetUrl(alias string) (string, error)
}

type Response struct {
	resp.Response
	Url   string `json:"url,omitempty"`
	Alias string `json:"alias,omitempty"`
}

func New(log *slog.Logger, get URLGet) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.url.get.New"
		log = log.With(
			slog.String("operation", op),
		)

		alias := chi.URLParam(r, "alias")
		if alias == "" {
			log.Error("Alias is empty")
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, "Alias is required")
			return
		}

		url, err := get.GetUrl(alias)
		if err != nil {

			if errors.Is(err, storage.ErrUrlNotFound) {
				log.Info("url not found", sl.Err(err))
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, resp.Error("url not found"))

				return
			}
			log.Error("failed to get url", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.Error("failed to get url"))

			return
		}
		
		responseOk(w, r, url)
	}
}

func responseOk(w http.ResponseWriter, r *http.Request, url string) {
	render.JSON(w, r, Response{
		Response: resp.Ok(),
		Url:      url,
	})
}
