FROM golang:alpine AS build

ADD . /app
WORKDIR /app

RUN go mod download
RUN go mod verify
RUN go build -o btclistings cmd/btclistings/main.go

FROM alpine:3.9 AS final
WORKDIR /usr/local/bin
COPY --from=build /app/btclistings ./

ENV PORT=80
EXPOSE $PORT
ENTRYPOINT ["/usr/local/bin/btclistings"]
