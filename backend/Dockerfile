FROM golang:1.25-alpine AS builder

WORKDIR /src
COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/bank-darah-api ./cmd/api

FROM alpine:3.20

WORKDIR /app
ENV PORT=8080
COPY --from=builder /out/bank-darah-api /app/bank-darah-api

EXPOSE 8080
CMD ["/app/bank-darah-api"]
