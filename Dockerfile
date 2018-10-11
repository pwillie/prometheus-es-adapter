# Build Stage
FROM golang:alpine AS build-stage

LABEL app="build-prometheus-es-adapter"
LABEL REPO="https://github.com/pwillie/prometheus-es-adapter"

RUN apk add -U -q --no-progress make git

ADD . /go/src/github.com/pwillie/prometheus-es-adapter
WORKDIR /go/src/github.com/pwillie/prometheus-es-adapter

RUN make build-alpine

# Final Stage
FROM alpine:latest

ENV USERID 789
ENV USERNAME pesa

RUN addgroup -g ${USERID} -S ${USERNAME} \
 && adduser -u ${USERID} -G ${USERNAME} -S ${USERNAME}

ARG GIT_COMMIT
ARG VERSION
LABEL REPO="https://github.com/pwillie/prometheus-es-adapter"
LABEL GIT_COMMIT=$GIT_COMMIT
LABEL VERSION=$VERSION

RUN apk add -U ca-certificates

COPY --from=build-stage /go/src/github.com/pwillie/prometheus-es-adapter/bin/prometheus-es-adapter /usr/local/bin/

USER ${USERNAME}

ENTRYPOINT [ "prometheus-es-adapter" ]
