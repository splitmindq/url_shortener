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
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"sync"
)

var (
	validate   *validator.Validate
	initOnce   sync.Once
	aliasRegex = regexp.MustCompile("^[a-zA-Z0-9_-]+$")
)

func initValidator() {
	validate = validator.New()
	validate.RegisterValidation("alphanum", func(fl validator.FieldLevel) bool {
		return aliasRegex.MatchString(fl.Field().String())
	})
}

func getValidator() *validator.Validate {
	initOnce.Do(initValidator)
	return validate
}

func isValidURL(u string) bool {
	parsed, err := url.Parse(u)
	if err != nil {
		return false
	}

	if parsed.Host == "" {
		return false
	}
	return true
}

func normalizeUrl(url string) string {
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		return "https://" + url
	}
	return url
}

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

func New(log *slog.Logger, urlSaver URLSaver, aliasLength int, maxAttempts int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.url.save.New"
		log = log.With(
			slog.String("operation", op),
		)
		var req Request
		err := render.DecodeJSON(r.Body, &req)
		if err != nil {
			log.Error("failed to parse request", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.Error("invalid request"))
			return
		}

		if err := getValidator().Struct(req); err != nil {
			var validationErrors validator.ValidationErrors
			errors.As(err, &validationErrors)

			log.Error("failed to validate request", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.ValidationError(validationErrors))
			return
		}

		alias := req.Alias

		if alias == "" {
			generatedAlias, err := generateUniqueAlias(urlSaver, log, aliasLength, maxAttempts)
			if err != nil {
				log.Error("failed to generate unique alias", sl.Err(err))
				render.Status(r, http.StatusInternalServerError)
				render.JSON(w, r, resp.Error("failed to generate unique alias"))
				return
			}
			alias = generatedAlias
			log.Info("generated unique alias", slog.String("alias", alias))
		}

		normalizedUrl := normalizeUrl(req.URL)
		if !isValidURL(normalizedUrl) {
			log.Error("invalid URL format", slog.String("url", normalizedUrl))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.Error("invalid URL format"))
			return
		}

		id, err := urlSaver.SaveURL(alias, normalizedUrl)
		if err != nil {
			if errors.Is(err, storage.ErrAliasExists) {
				log.Error("Alias already exist", slog.String("url", req.URL))
				render.Status(r, http.StatusConflict)
				render.JSON(w, r, resp.Error("alias already exist"))
				return
			}
			log.Error("failed to save url", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
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

func generateUniqueAlias(urlSaver URLSaver, log *slog.Logger, aliasLength int, maxAttempts int) (string, error) {
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
