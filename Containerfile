FROM golang:1.22

COPY . /go/src/github.com/led0nk/guestbook

WORKDIR /go/src/github.com/led0nk/guestbook

RUN CGO_ENABLED=0 go build -o bin/guestbook cmd/server/main.go

FROM scratch

COPY --from=0 bin/guestbook /guestbook

EXPOSE 8080

CMD ["/guestbook"]
