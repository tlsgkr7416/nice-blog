FROM golang:alpine AS builder
ENV GO111MODULE=on \
    CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=arm64

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY web ./web
COPY main.go ./
RUN go build -o main .

WORKDIR /dist
RUN cp /build/main .

FROM scratch
COPY --from=builder /lib/ld-musl-aarch64.so.1 /lib/ld-musl-aarch64.so.1
COPY --from=builder /bin/busybox /bin/busybox
COPY --from=builder /dist/main .

ENTRYPOINT ["/main"]
