package save

import (
	resp "URL-Shortener/internal/lib/api/response"
	"URL-Shortener/internal/lib/logger/sl"
	"URL-Shortener/internal/lib/random"
	"URL-Shortener/internal/storage"
	"errors"
	"github.com/go-chi/render"
	"github.com/go-playground/validator/v10"
	"log/slog"
	"net/http"
	"strconv"
)

// TODO: conf
const aliasLength = 6

type URLSaver interface {
	SaveURL(alias string, urlToSave string) (int64, error)
}

type Request struct {
	Alias string `json:"alias,omitempty"`
	URL   string `json:"url" validate:"required"`
}

type Response struct {
	resp.Response
	Alias string `json:"alias,omitempty"`
}

func New(log *slog.Logger, urlSaver URLSaver) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.url.save.New"
		log.With(
			slog.String("operation", op),
		)
		var req Request

		err := render.DecodeJSON(r.Body, &req)
		if err != nil {
			log.Error("failed to parse request", sl.Err(err))

			render.JSON(w, r, resp.Error("failed to decode request"))

			return
		}
		log.Info("request body decoded", slog.Any("request", req))

		if err := validator.New().Struct(req); err != nil {
			var validateErr validator.ValidationErrors
			errors.As(err, &validateErr)

			log.Error("failed to validate request", sl.Err(err))

			render.JSON(w, r, resp.ValidationError(validateErr))

			return
		}

		alias := req.Alias
		//todo if equal to existing
		if alias == "" {
			alias = random.NewRandomAlias(aliasLength)
		}

		//todo
		//alias <->  url, alias must be unique to url => except alias already exist error
		id, err := urlSaver.SaveURL(alias, req.URL)
		if err != nil {
			if errors.Is(err, storage.ErrAliasExists) {
				log.Error("Alias already exist", slog.String("url", req.URL))
				render.JSON(w, r, resp.Error("url already exist"))
				return
			}
			log.Error("failed to save url", sl.Err(err))

			render.JSON(w, r, resp.Error("failed to save url"))

			return
		}
		log.Info("saved url", slog.String("url", req.URL), slog.String("id", strconv.FormatInt(id, 10)))

		responseOk(w, r, alias)
	}

}

func responseOk(w http.ResponseWriter, r *http.Request, alias string) {
	render.JSON(w, r, Response{
		Response: resp.Ok(),
		Alias:    alias,
	})
}
