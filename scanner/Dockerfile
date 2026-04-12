FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod go.sum* ./
RUN go mod download
COPY . .
RUN go build -o kestrel-scanner ./cmd/main.go

FROM alpine:3.19
RUN apk add --no-cache ca-certificates curl

# Install pluto
RUN curl -sSL https://github.com/FairwindsOps/pluto/releases/latest/download/pluto_linux_amd64.tar.gz \
    | tar -xz -C /usr/local/bin pluto

# Install polaris
RUN curl -sSL https://github.com/FairwindsOps/polaris/releases/latest/download/polaris_linux_amd64.tar.gz \
    | tar -xz -C /usr/local/bin polaris

# Install kube-score
RUN curl -sSL https://github.com/zegl/kube-score/releases/latest/download/kube-score_linux_amd64.tar.gz \
    | tar -xz -C /usr/local/bin kube-score

COPY --from=builder /app/kestrel-scanner /usr/local/bin/kestrel-scanner

EXPOSE 8090
ENTRYPOINT ["kestrel-scanner"]
