FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY cmd ./cmd
COPY internal ./internal
COPY web ./web
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/ ./cmd/vuc ./cmd/migrate ./cmd/seed ./cmd/importdata

FROM alpine:3.21
RUN addgroup -S vuc && adduser -S -G vuc vuc
WORKDIR /app
COPY --from=build /out/ /usr/local/bin/
RUN mkdir /app/data && chown vuc:vuc /app/data
USER vuc
EXPOSE 8080
VOLUME ["/app/data"]
ENTRYPOINT ["vuc"]
CMD ["-addr", ":8080", "-data", "/app/data/vuc.json"]
