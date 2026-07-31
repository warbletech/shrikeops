FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod go.sum* ./
RUN go mod download
COPY . .
RUN go build -o shrikeops-scanner ./cmd/main.go

FROM alpine:3.19
RUN apk add --no-cache ca-certificates curl tar

# Install pluto (pinned v5.19.4)
RUN curl -fL -o /tmp/pluto.tar.gz \
    "https://github.com/FairwindsOps/pluto/releases/download/v5.19.4/pluto_5.19.4_linux_amd64.tar.gz" \
    && tar -xz -C /usr/local/bin -f /tmp/pluto.tar.gz pluto \
    && rm /tmp/pluto.tar.gz

# Install polaris (pinned v9.6.1)
RUN curl -fL -o /tmp/polaris.tar.gz \
    "https://github.com/FairwindsOps/polaris/releases/download/9.6.1/polaris_linux_amd64.tar.gz" \
    && tar -xz -C /usr/local/bin -f /tmp/polaris.tar.gz polaris \
    && rm /tmp/polaris.tar.gz

# Install kube-score (pinned v1.19.0)
RUN curl -fL -o /tmp/kube-score.tar.gz \
    "https://github.com/zegl/kube-score/releases/download/v1.19.0/kube-score_1.19.0_linux_amd64.tar.gz" \
    && tar -xz -C /usr/local/bin -f /tmp/kube-score.tar.gz kube-score \
    && rm /tmp/kube-score.tar.gz

COPY --from=builder /app/shrikeops-scanner /usr/local/bin/shrikeops-scanner

EXPOSE 8090
ENTRYPOINT ["shrikeops-scanner"]
