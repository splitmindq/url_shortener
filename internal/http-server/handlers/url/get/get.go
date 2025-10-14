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

type Request struct {
	Alias string `json:"alias"`
}

type Response struct {
	resp.Response
	Url string `json:"url,omitempty"`
}

func New(log *slog.Logger, get URLGet) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.url.get.New"
		log = log.With(
			slog.String("operation", op),
		)

		//todo log ->
		alias := chi.URLParam(r, "alias")
		if alias == "" {
			log.Info("Alias is empty")

			render.JSON(w, r, "Empty Alias")

			return

		}

		var req Request
		req.Alias = alias

		url, err := get.GetUrl(req.Alias)
		if err != nil {

			if errors.Is(err, storage.ErrUrlNotFound) {
				log.Info("url not found", sl.Err(err))

				render.JSON(w, r, resp.Error("url not found"))

				return
			}
			log.Error("failed to get url", sl.Err(err))

			render.JSON(w, r, resp.Error("failed to get url"))

			return
		}
		if url == "" {
			log.Error("url not found", sl.Err(err))

			render.JSON(w, r, resp.Error("url not found"))

			return
		}
		log.Info("url found", slog.String("url", url))

		responseOk(w, r, url)
	})
}

func responseOk(w http.ResponseWriter, r *http.Request, url string) {
	render.JSON(w, r, Response{
		Response: resp.Ok(),
		Url:      url,
	})
}
