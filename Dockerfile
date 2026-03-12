# syntax=docker/dockerfile:1.7

FROM --platform=$BUILDPLATFORM golang:1.26-alpine AS build

ARG TARGETOS
ARG TARGETARCH

WORKDIR /src

COPY go.mod ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -trimpath -ldflags="-s -w" -o /out/miniflux-cli .

FROM alpine:3.21

RUN adduser -D -u 10001 appuser

WORKDIR /app

COPY --from=build /out/miniflux-cli /usr/local/bin/miniflux-cli

USER appuser

ENTRYPOINT ["miniflux-cli"]
CMD ["me"]
