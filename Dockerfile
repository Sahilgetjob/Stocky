FROM golang:1.23 AS builder

WORKDIR /app
COPY go.mod .
RUN go mod download

COPY . .
RUN go mod tidy
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /stocky ./cmd/server

FROM gcr.io/distroless/base-debian12
WORKDIR /
ENV TZ=Asia/Kolkata
COPY --from=builder /stocky /stocky
COPY --from=builder /app/.env /.env
EXPOSE 8080
USER 65532:65532
ENTRYPOINT ["/stocky"]
