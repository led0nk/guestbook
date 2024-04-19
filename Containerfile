FROM golang:1.22

COPY . /go/src/github.com/led0nk/guestbook

WORKDIR /go/src/github.com/led0nk/guestbook

RUN make build

FROM scratch

COPY --from=0 cmd/server/guestbook /guestbook

EXPOSE 8080

CMD ["/guestbook"]
