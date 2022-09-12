# syntax=docker/dockerfile:experimental
FROM golang:1.19.1-alpine3.16 AS build

ENV CGO_ENABLED=0 \
    LANG=C.UTF-8

WORKDIR /src

COPY go.mod go.sum ./
COPY *.go ./
RUN go build -o /cr ./

FROM scratch 

ENV APP_UID=10000 \
    APP_GID=10000

COPY --from=build /cr /

USER ${APP_UID}:${APP_GID}

CMD ["/cr"]
