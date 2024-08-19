package redirect

import (
	"errors"
	"log/slog"
	"net/http"
	"server/internal/lib/api/response"
	"server/internal/lib/logger/sl"
	"server/internal/storage"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/render"
)

type URLGetter interface {
	GetURL(alias string) (string, error)
}

func New(log *slog.Logger, urlGetter URLGetter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.url.redirect.New"

		log = log.With(
			slog.String("op", op),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		alias := chi.URLParam(r, "alias")

		if alias == "" {
			log.Info("alias is empty")
			render.JSON(w, r, response.Error("alias is empty"))
			return
		}

		resURL, err := urlGetter.GetURL(alias)
		if errors.Is(err, storage.ErrURLNotFound) {
			render.JSON(w, r, response.Error("not found"))
			return
		}

		if err != nil {
			log.Error("failed to get url", sl.Err(err))
			render.JSON(w, r, "internal error")
			return
		}

		log.Info("got url", slog.String("url", resURL))

		http.Redirect(w, r, resURL, http.StatusFound)

	}
}
