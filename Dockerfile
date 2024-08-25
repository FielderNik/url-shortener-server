FROM golang:1.23

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download && go mod verify

COPY . .

WORKDIR /app

# RUN go build -o /app/server/
RUN go build -o server ./cmd/url-shortener

CMD ["/app/server"]


EXPOSE 80