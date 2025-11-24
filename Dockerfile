FROM golang:1.24
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
ENV SSO_CONFIG_PATH=./config/config.yaml
EXPOSE 50051
CMD ["sh", "-c", "go run ./cmd/migrator && go run ./cmd/sso"]