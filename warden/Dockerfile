FROM --platform=$BUILDPLATFORM golang:1.26-alpine AS builder

RUN apk --no-cache add ca-certificates
RUN apk add --no-cache tzdata

WORKDIR /app

COPY go.mod ./
COPY go.sum ./
RUN go mod download

COPY . ./
ARG TARGETOS
ARG TARGETARCH
RUN GOOS=$TARGETOS GOARCH=$TARGETARCH go build -o /server

FROM alpine:3.19

# ca-certificates ships the standard CA bundle at
# /etc/ssl/certs/ca-certificates.crt so Go's crypto/x509 can validate
# TLS handshakes against public origins (e.g., OIDC discovery URLs).
# Without this, outbound HTTPS from the vault-server binary fails on
# TLS verify even though wget in the same image works, because Go
# probes for the bundle differently than OpenSSL/LibreSSL does.
RUN apk --no-cache add ca-certificates

COPY --from=builder /server /server

COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
ENV TZ=America/Los_Angeles

ENTRYPOINT ["/server"]
