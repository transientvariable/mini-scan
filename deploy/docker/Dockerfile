# Install dependencies
# ========================================
FROM golang:1.24-bookworm AS deps

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download && go mod verify

# Final image build
# ========================================
FROM golang:1.24-bookworm AS builder

WORKDIR /src

COPY --from=deps /go/pkg /go/pkg
COPY . .
RUN go build -ldflags="-w -s" -o scanner .

# Final image build
# ========================================
FROM debian:bookworm-slim

ARG BUILD_DATE=${BUILD_DATE}
ARG APPLICATION_NAME=scanner
ARG WORK_DIR=/app
ARG USER=scanner
ARG GROUP=scanner

LABEL org.opencontainers.image.created=$BUILD_DATE \
      org.opencontainers.image.description="mini-scan CLI" \
      org.opencontainers.image.source=https://github.com/transientvariable/mini-scan \
      org.opencontainers.image.title=$APPLICATION_NAME

WORKDIR ${WORK_DIR}

# Create a non-root user and group
RUN groupadd -r ${USER} && useradd -r -g ${GROUP} ${USER}
COPY --chown=$USER:$USER --from=builder /src/scanner .
COPY --chown=$USER:$USER deploy/migration ./deploy/migration
COPY --chown=$USER:$USER deploy/repository ./deploy/repository
COPY --chown=$USER:$USER application.yaml application.yaml

# Entrypoint
# ========================================

# Switch to the non-root user
USER ${USER}

CMD ["/app/scanner"]