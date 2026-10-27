FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY go.mod go.sum* ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o shrikeops-scanner ./cmd/main.go \
 && CGO_ENABLED=0 go build -o shrikeops ./cmd/cli

FROM alpine:3.19
ARG TARGETARCH=amd64
# Scanner tool versions — keep in sync with README "Requirements" and
# internal/engine/testdata/README.md (parsers are tested against these outputs).
ARG PLUTO_VERSION=5.24.4
ARG POLARIS_VERSION=10.2.5
ARG KUBE_SCORE_VERSION=1.20.0
RUN apk add --no-cache ca-certificates curl tar \
 && curl -fsSL "https://github.com/FairwindsOps/pluto/releases/download/v${PLUTO_VERSION}/pluto_${PLUTO_VERSION}_linux_${TARGETARCH}.tar.gz" \
    | tar -xz -C /usr/local/bin pluto \
 && curl -fsSL "https://github.com/FairwindsOps/polaris/releases/download/v${POLARIS_VERSION}/polaris_${POLARIS_VERSION}_linux_${TARGETARCH}.tar.gz" \
    | tar -xz -C /usr/local/bin polaris \
 && curl -fsSL "https://github.com/zegl/kube-score/releases/download/v${KUBE_SCORE_VERSION}/kube-score_${KUBE_SCORE_VERSION}_linux_${TARGETARCH}.tar.gz" \
    | tar -xz -C /usr/local/bin kube-score \
 && pluto version && polaris version && kube-score version

COPY --from=builder /app/shrikeops-scanner /usr/local/bin/shrikeops-scanner
# CLI is included too: docker run --entrypoint shrikeops IMAGE scan - < manifest.yaml
COPY --from=builder /app/shrikeops /usr/local/bin/shrikeops

EXPOSE 8090
ENTRYPOINT ["shrikeops-scanner"]
