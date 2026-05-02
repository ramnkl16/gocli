# syntax=docker/dockerfile:1

FROM golang:1.26-bookworm AS build
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG VERSION=0.0.0-dev
ARG COMMIT=none
ARG DATE=unknown

RUN CGO_ENABLED=0 GOOS=linux go build -trimpath \
    -ldflags="-s -w -X main.version=${VERSION} -X main.commit=${COMMIT} -X main.date=${DATE}" \
    -o /out/gocli .

FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /
COPY --from=build --chown=nonroot:nonroot /out/gocli /gocli
USER nonroot:nonroot
ENTRYPOINT ["/gocli"]
