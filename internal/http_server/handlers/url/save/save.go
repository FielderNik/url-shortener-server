package save

import (
	"errors"
	"io"
	"log/slog"
	"net/http"
	"server/internal/lib/api/response"
	"server/internal/lib/logger/sl"
	"server/internal/lib/random"
	"server/internal/storage"

	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/render"
	"github.com/go-playground/validator"
)

const aliasLength = 6

type Request struct {
    URL   string `json:"url" validate:"required,url"`
    Alias string `json:"alias,omitempty"`
}

type Response struct {
    response.Response
    Alias string `json:"alias"`
}

type URLSaver interface {
    SaveURL(URL, alias string) (int64, error)
}

func New(log *slog.Logger, urlSaver URLSaver) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        const op = "handlers.url.save.New"


        // Добавляем к текущму объекту логгера поля op и request_id
        // Они могут очень упростить нам жизнь в будущем
        log = log.With(
            slog.String("op", op),
            slog.String("request_id", middleware.GetReqID(r.Context())),
        )

        // Создаем объект запроса и анмаршаллим в него запрос
        var req Request

        err := render.DecodeJSON(r.Body, &req)
        if errors.Is(err, io.EOF) {
            // Такую ошибку встретим, если получили запрос с пустым телом
            // Обработаем её отдельно
            log.Error("request body is empty")

            render.JSON(w, r, response.Error("request body is empty"))

            return
        }
        if err != nil {
            log.Error("failed to decode request body", sl.Err(err))

            render.JSON(w, r, response.Error("failed to decode request"))

            return
        }

        if err := validator.New().Struct(req); err != nil {
            // Приводим ошибку к типу ошибки валидации
            validateErr := err.(validator.ValidationErrors)
        
            log.Error("invalid request", sl.Err(err))
        
            render.JSON(w, r, response.Error(validateErr.Error()))
        
            return
        }

        // Лучше больше логов, чем меньше - лишнее мы легко сможем почистить,
        // при необходимости. А вот недостающую информацию мы уже не получим.
        log.Info("request body decoded", slog.Any("req", req))

        alias := req.Alias
        if alias == "" {
            alias = random.NewRandomString(aliasLength)
        }

        id, err := urlSaver.SaveURL(req.URL, alias)
        if errors.Is(err, storage.ErrURLExists) {
            // Отдельно обрабатываем ситуацию,
            // когда запись с таким Alias уже существует
            log.Info("url already exists", slog.String("url", req.URL))

            render.JSON(w, r, response.Error("url already exists"))

            return
        }
        if err != nil {
            log.Error("failed to add url", sl.Err(err))

            render.JSON(w, r, response.Error("failed to add url"))

            return
        }

        log.Info("url added", slog.Int64("id", id))

        responseOK(w, r, alias)
    }
}

func responseOK(w http.ResponseWriter, r *http.Request, alias string) {
    render.JSON(w, r, Response{
        Response: response.OK(),
        Alias:    alias,
    })
}
