FROM golang:1.25-bookworm AS build
WORKDIR /src
COPY komari/go.mod komari/go.sum ./
ARG GOPROXY=https://goproxy.cn,direct
ENV GOPROXY=${GOPROXY}
RUN go mod download -x
COPY komari/ ./
# Metadata is configuration schema only; no web bundle is embedded.
COPY komari-web/komari-theme.json ./web/public/defaultTheme/komari-theme.json
RUN CGO_ENABLED=1 go build -tags api_only -trimpath -o /out/komari .

FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates tzdata && rm -rf /var/lib/apt/lists/*
WORKDIR /app
COPY --from=build /out/komari /usr/local/bin/komari
ENV GIN_MODE=release KOMARI_LISTEN=0.0.0.0:25774
VOLUME ["/app/data"]
EXPOSE 25774
CMD ["komari", "server"]
