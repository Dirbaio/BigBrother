FROM golang:1.9 AS build
COPY . /go/src/github.com/Dirbaio/BigBrother
RUN go get github.com/Dirbaio/BigBrother

FROM node:8 AS frontend-build
COPY frontend /frontend
RUN cd /frontend && \
    npm install && \
    npm run build

FROM jrottenberg/ffmpeg:3.3
COPY --from=build /go/bin/BigBrother /usr/local/bin/
RUN mkdir -p /app
COPY --from=frontend-build /frontend/dist /app/static
WORKDIR /app
ENTRYPOINT ["/usr/local/bin/BigBrother"]
