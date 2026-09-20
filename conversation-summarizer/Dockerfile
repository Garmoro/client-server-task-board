FROM golang:1.25-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags='-s -w' -o /out/conversation-summarizer ./cmd/app

FROM alpine:3.21
RUN addgroup -S app && adduser -S -G app app
WORKDIR /app
COPY --from=build /out/conversation-summarizer /app/conversation-summarizer
COPY migrations /app/migrations
USER app
EXPOSE 8091
ENTRYPOINT ["/app/conversation-summarizer"]