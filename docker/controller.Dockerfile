ARG BUILDPLATFORM
FROM --platform=$BUILDPLATFORM golang:1.23-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download && go mod verify

COPY api api
COPY cmd cmd
COPY internal internal
# Build the controller binary
ARG TARGETARCH
ARG LDFLAGS
RUN CGO_ENABLED=0 GOOS=linux GOARCH=${TARGETARCH} go build -a -ldflags "$LDFLAGS" -o bin/controller cmd/controller/main.go

# Use distroless for minimal attack surface
FROM gcr.io/distroless/static:nonroot

WORKDIR /

COPY --from=builder /app/bin/controller /controller

USER 65532:65532

LABEL org.opencontainers.image.source=https://github.com/den-vasyliev/agentregistry-inventory
LABEL org.opencontainers.image.description="Agent Registry Controller"
LABEL org.opencontainers.image.authors="Agent Registry Creators"

ENTRYPOINT ["/controller"]
