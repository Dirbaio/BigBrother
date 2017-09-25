FROM golang:1.9 AS build
COPY . /go/src/github.com/Dirbaio/BigBrother
RUN go get github.com/Dirbaio/BigBrother

FROM jrottenberg/ffmpeg:3.3
COPY --from=build /go/bin/BigBrother /usr/local/bin/
RUN mkdir -p /app
COPY static /app/static
WORKDIR /app
ENTRYPOINT ["/usr/local/bin/BigBrother"]
