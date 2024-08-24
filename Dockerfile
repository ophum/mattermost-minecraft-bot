FROM golang:1.22.2 AS builder

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 go build -o mattermost-minecraft-bot main.go bot.go

FROM gcr.io/distroless/static-debian12:latest

WORKDIR /app

COPY --from=builder /app/mattermost-minecraft-bot .

ENTRYPOINT ["/app/mattermost-minecraft-bot"]