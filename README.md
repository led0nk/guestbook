# Guestbook

Welcome to the Guestbook! The Guestbook is designed to manage guestbook-entries for events e.g. a wedding or a birthday-party.
The application is mainly written in [Go](https://go.dev/), instrumented via [OpenTelemetry](https://opentelemetry.io/) and presented by the [Grafana-LGTM-Stack](https://github.com/grafana/docker-otel-lgtm) for improved observability.

![1715542600-guestbook.png](assets/imgs/1715542600-guestbook.png)

## Get started

You can easily get started via docker:
```shell
docker run -it -p 8080:8080 --rm ghcr.io/led0nk/guestbook:latest
```


## Appliable flags:

| flag        | default          | function                          |
| ----------- | ---------------- | --------------------------------- |
| `-addr`     | `localhost:8080` | server address                    |
| `-grpcaddr` | <nil>            | grpc address, e.g. localhost:4317 |
| `-db`       | `file://testdata`  | path to database                  |
| `-env`      | `testdata/.env`    | path to .env-file                 |
| `-domain`   | `127.0.0.1`        | given domain for cookies/mail     |
| `-loglevel` | `INFO`             | define the level for logs         |

## Configuration

In order to provide the guestbook with a working authentication system, which uses E-Mail validation, you need to pre-configure some variables in `.env`.
It should at least contain the following:
```dotenv
TOKENSECRET="yoursecretofchoice"
EMAIL="youremail@domain.com"
SMTPPW="dontforgettosetupyoursmtppw"
HOST="smtp.domain.com"
PORT="587"
```

**Remember:** You have to set up your smtp-password for your email-provider.

## Important

The application should be defined as "pre-alpha" due to the lack of frontend-variation and code quality.
There is much room for improvement e.g. creating multiple account-validation methods, reworking some frontend parts, writing more efficient code and many more...

