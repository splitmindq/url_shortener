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
const maxAttempts = 10

type URLSaver interface {
	SaveURL(alias string, urlToSave string) (int64, error)
	AliasExists(alias string) (bool, error)
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
		log = log.With(
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

		if alias == "" {
			generatedAlias, err := generateUniqueAlias(urlSaver, log)
			if err != nil {
				log.Error("failed to generate unique alias", sl.Err(err))
				render.JSON(w, r, resp.Error("failed to generate unique alias"))
				return
			}
			alias = generatedAlias
			log.Info("generated unique alias", slog.String("alias", alias))
		}

		id, err := urlSaver.SaveURL(alias, req.URL)
		if err != nil {
			if errors.Is(err, storage.ErrAliasExists) {
				log.Error("Alias already exist", slog.String("url", req.URL))
				render.JSON(w, r, resp.Error("alias already exist"))
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

func generateUniqueAlias(urlSaver URLSaver, log *slog.Logger) (string, error) {
	for i := 0; i < maxAttempts; i++ {
		alias := random.NewRandomAlias(aliasLength)

		exists, err := urlSaver.AliasExists(alias)
		if err != nil {
			return "", err
		}

		if !exists {
			return alias, nil
		}

		log.Debug("alias collision, generating new one",
			slog.String("alias", alias),
			slog.Int("attempt", i+1),
		)
	}

	return "", errors.New("failed to generate unique alias after maximum attempts")
}
