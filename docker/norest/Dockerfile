FROM golang:1.26-alpine AS builder

RUN apk add --no-cache git

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /out/norest-api ./cmd/api
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /out/norest-worker ./cmd/worker
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /out/norest-init-system-account ./cmd/init-system-account

FROM gcr.io/distroless/static-debian12

COPY --from=builder /out/norest-api /norest-api
COPY --from=builder /out/norest-worker /norest-worker
COPY --from=builder /out/norest-init-system-account /norest-init-system-account

USER nonroot:nonroot

ENTRYPOINT ["/norest-api"]
