package redirect

import (
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

func New(log *slog.Logger, get URLGet) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.redirect.New"
		log = log.With(slog.String("operation", op))

		alias := chi.URLParam(r, "alias")
		if alias == "" {
			log.Info("alias is empty")
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, "Empty Alias")

			return
		}

		resUrl, err := get.GetUrl(alias)
		if err != nil {
			if errors.Is(err, storage.ErrUrlNotFound) {
				log.Info("url not found", "alias", alias)
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, "Url not found")
				return
			}
			log.Error(err.Error(), "url", resUrl)
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, "Internal server error")
			return
		}

		http.Redirect(w, r, resUrl, http.StatusFound)
	}
}
