FROM node:20-alpine AS web-builder

WORKDIR /web

COPY web/package.json web/package-lock.json ./
RUN npm ci

COPY web/ ./
RUN npm run build

FROM golang:1.23-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/teamtimebot ./cmd/teamtimebot
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/migrate ./cmd/migrate
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/web ./cmd/web

FROM gcr.io/distroless/static-debian12

ENV TZ=UTC

COPY --from=builder /out/teamtimebot /teamtimebot
COPY --from=builder /out/migrate /migrate
COPY --from=builder /out/web /web
COPY --from=builder /app/db/migrations /migrations
COPY --from=web-builder /web/dist /web-static

ENTRYPOINT ["/teamtimebot"]
