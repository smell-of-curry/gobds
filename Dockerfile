FROM golang:1.24-alpine AS builder

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o gobds ./main.go

FROM alpine:latest AS runner

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

COPY --from=builder /build/gobds .

EXPOSE 19132/udp
CMD ["./gobds"]
