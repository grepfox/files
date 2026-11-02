FROM golang:1.23-alpine AS build
WORKDIR /src
COPY . .
ARG SECRET
ARG HOST
ARG PASS
ENV CGO_ENABLED=0 GOOS=linux GOARCH=amd64
RUN go build -trimpath -ldflags "-s -w -X main.secret=$SECRET -X main.pass=$PASS" -o /server ./server
RUN go build -trimpath -ldflags "-s -w -X main.secret=$SECRET -X main.host=$HOST" -o /upload ./up

FROM scratch
COPY --from=build /server /upload /
VOLUME /data
EXPOSE 8080
ENTRYPOINT ["/server"]
