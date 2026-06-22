FROM golang:1.25-alpine AS builder

RUN apk add --no-cache gcc musl-dev

WORKDIR /build

COPY . .
RUN go mod tidy && CGO_ENABLED=1 GOOS=linux go build -o /build/server ./cmd/server && CGO_ENABLED=1 GOOS=linux go build -o /build/bulk-import ./cmd/bulk-import

FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=builder /build/server .
COPY --from=builder /build/bulk-import .

EXPOSE 8080

CMD ["./server"]
