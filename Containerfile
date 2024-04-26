FROM golang:1.22

LABEL org.opencontainers.image.source https://github.com/led0nk/guestbook

COPY . /go/src/github.com/led0nk/guestbook

WORKDIR /go/src/github.com/led0nk/guestbook

RUN CGO_ENABLED=0 go build -v -o /guestbook cmd/server/main.go

FROM scratch

COPY --from=0 /guestbook /guestbook

EXPOSE 8080

CMD ["/guestbook"]
