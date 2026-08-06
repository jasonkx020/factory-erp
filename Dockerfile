# 多阶段构建 ERP API
FROM golang:1.24-alpine AS build
WORKDIR /src
ENV GOTOOLCHAIN=auto
RUN apk add --no-cache git ca-certificates
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/erp-api ./cmd/erp-api

FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata wget
WORKDIR /app
COPY --from=build /out/erp-api /app/erp-api
COPY configs/erp.prod.example.yaml /app/configs/erp.yaml
RUN mkdir -p /app/data
ENV ERP_CONFIG=/app/configs/erp.yaml
EXPOSE 18080
HEALTHCHECK --interval=30s --timeout=5s --start-period=20s --retries=3 \
  CMD wget -qO- http://127.0.0.1:18080/api/v1/live || exit 1
ENTRYPOINT ["/app/erp-api"]
CMD ["-config", "/app/configs/erp.yaml"]
