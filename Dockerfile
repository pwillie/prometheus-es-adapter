# Build Stage
FROM golang:alpine AS build-stage

LABEL app="build-prometheus-es-adapter"
LABEL REPO="https://github.com/ycyr/prometheus-es-adapter"

ENV GOROOT=/usr/local/go \
    GOPATH=/gopath \
    GOBIN=/gopath/bin \
    PROJPATH=/gopath/src/github.com/ycyr/prometheus-es-adapter

RUN apk add -U -q --no-progress build-base git glide

ADD . /gopath/src/github.com/ycyr/prometheus-es-adapter
WORKDIR /gopath/src/github.com/ycyr/prometheus-es-adapter

RUN make get-deps build-alpine

# Final Stage
FROM alpine:latest

ARG GIT_COMMIT
ARG VERSION
LABEL REPO="https://github.com/ycyr/prometheus-es-adapter"
LABEL GIT_COMMIT=$GIT_COMMIT
LABEL VERSION=$VERSION

# Because of https://github.com/docker/docker/issues/14914
ENV PATH=$PATH:/opt/prometheus-es-adapter/bin

WORKDIR /opt/prometheus-es-adapter/bin

COPY --from=build-stage /gopath/src/github.com/ycyr/prometheus-es-adapter/bin/prometheus-es-adapter /opt/prometheus-es-adapter/bin/
RUN chmod +x /opt/prometheus-es-adapter/bin/prometheus-es-adapter

ENTRYPOINT [ "/opt/prometheus-es-adapter/bin/prometheus-es-adapter" ]
