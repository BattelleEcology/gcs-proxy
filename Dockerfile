FROM golang:1.27.2 AS builder
WORKDIR /myapp
COPY go.mod go.sum ./
RUN go mod download
COPY . .
# Declared here, not at the top, so a new version doesn't invalidate the
# module download layer.
ARG VERSION=SNAPSHOT
RUN CGO_ENABLED=0 go build -ldflags "-X main.version=${VERSION}" -o gcs-proxy .

# alpine already provides CA certificates, /etc/nsswitch.conf and the nobody
# user.
FROM alpine:3.24.2 AS final
COPY --from=builder /myapp/gcs-proxy /
# The gops agent writes its port file under $HOME/.config by default, and
# nobody's home (/) isn't writable.
ENV GOPS_CONFIG_DIR=/tmp/gops
USER nobody
ENTRYPOINT ["/gcs-proxy"]
