FROM golang:1.24 AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/sso ./cmd/sso \
 && CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/migrator ./cmd/migrator

FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /app
COPY --from=build /out/sso /out/migrator ./
COPY migrations ./migrations
COPY config ./config
ENV SSO_CONFIG_PATH=/app/config/config.yaml
EXPOSE 50051
ENTRYPOINT ["/app/sso"]
