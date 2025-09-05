FROM golang:1.23
WORKDIR /app
COPY . /app
RUN go mod download
ENV CGO_ENABLED=0
COPY ./configs/bot.yaml /app/configs/bot.yaml
CMD ["go", "run", "./cmd/bot/main.go"]
