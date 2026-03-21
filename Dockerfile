# ── Stage 1: Build the SPA ────────────────────────────────────────────────────
FROM node:22-alpine AS frontend

WORKDIR /build/web
COPY web/package.json web/package-lock.json ./
RUN npm ci
COPY web/ ./
RUN npm run build

# ── Stage 2: Build the Go server (with embedded frontend) ────────────────────
FROM golang:1.25-alpine AS backend

WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download

COPY . .
COPY --from=frontend /build/web/dist ./cmd/server/web/dist

RUN CGO_ENABLED=0 go build -tags frontend -o /server ./cmd/server/
RUN CGO_ENABLED=0 go build -o /migrate ./cmd/migrate/

# ── Stage 3: Minimal runtime image ───────────────────────────────────────────
FROM gcr.io/distroless/static-debian12

COPY --from=backend /server /server
COPY --from=backend /migrate /migrate

EXPOSE 8080
ENTRYPOINT ["/server"]
