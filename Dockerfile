FROM golang:1.27 AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/i-pro-camera-backup ./cmd/i-pro-camera-backup

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/i-pro-camera-backup /i-pro-camera-backup
ENTRYPOINT ["/i-pro-camera-backup"]
