FROM golang:1.27 AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/i-pro-camera-downloader ./cmd/i-pro-camera-downloader

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/i-pro-camera-downloader /i-pro-camera-downloader
ENTRYPOINT ["/i-pro-camera-downloader"]
