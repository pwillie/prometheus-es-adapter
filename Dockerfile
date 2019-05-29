# Build Stage
FROM golang:1.12 AS build-stage

WORKDIR /src
ADD . .

RUN make test && make build

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

COPY --from=build-stage /src/release/linux/amd64/prometheus-es-adapter /usr/local/bin/

USER ${USERNAME}

ENTRYPOINT [ "prometheus-es-adapter" ]
