FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod ./
COPY cmd ./cmd
COPY internal ./internal
COPY web ./web
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/vuc ./cmd/vuc

FROM alpine:3.21
RUN addgroup -S vuc && adduser -S -G vuc vuc
WORKDIR /app
COPY --from=build /out/vuc /usr/local/bin/vuc
RUN mkdir /app/data && chown vuc:vuc /app/data
USER vuc
EXPOSE 8080
VOLUME ["/app/data"]
ENTRYPOINT ["vuc", "-addr", ":8080", "-data", "/app/data/vuc.json"]
