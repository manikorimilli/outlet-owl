# OutletOwl for the demo deployment (ADR-0010): one image with the Go API and
# the built web app, which the server serves from the same address
# (ADR-0002). Build:
#   docker build --build-arg VERSION=$(git describe --tags --always) -t outletowl .
# DEMO_PREFILL=true prefills the sign-in form and shows the demo note; leave
# it false for anything but the public demo.

FROM node:24-slim AS web
WORKDIR /web
RUN corepack enable
COPY web/package.json web/pnpm-lock.yaml web/pnpm-workspace.yaml ./
RUN pnpm install --frozen-lockfile
COPY web/ ./
ARG DEMO_PREFILL=false
RUN VITE_DEMO_PREFILL=${DEMO_PREFILL} pnpm run build

FROM golang:1.26.9 AS api
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY cmd ./cmd
COPY internal ./internal
COPY prompts ./prompts
ARG VERSION=dev
RUN CGO_ENABLED=0 go build -trimpath \
    -ldflags="-s -w -X main.version=${VERSION}" \
    -o /out/api ./cmd/api

# distroless: no shell and no package manager, runs as a non-root user. The
# time zone data is compiled into the binary (time/tzdata in cmd/api).
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=api /out/api /app/api
COPY --from=web /web/dist /app/web
ENV HOST=0.0.0.0 \
    PORT=8080 \
    WEB_DIST=/app/web
EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/app/api"]
