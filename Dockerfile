ARG GO_IMAGE=golang:1.23-alpine
ARG RUNTIME_IMAGE=alpine:3.20

FROM ${GO_IMAGE} AS builder
RUN sh -lc 'command -v apk >/dev/null 2>&1 && apk add --no-cache git ca-certificates || true'
WORKDIR /src
COPY go.mod ./
COPY . .
RUN sh -lc 'export PATH=/usr/local/go/bin:$PATH && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o /out/contrib-sync ./cmd/contrib-sync'

FROM ${RUNTIME_IMAGE}
RUN sh -lc 'command -v apk >/dev/null 2>&1 && apk add --no-cache git ca-certificates tzdata || true'
COPY --from=builder /out/contrib-sync /usr/local/bin/contrib-sync
ENTRYPOINT ["contrib-sync"]
CMD ["help"]
