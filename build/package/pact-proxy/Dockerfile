FROM golang:1.22-alpine AS build-env

ARG APPNAME
ENV GO111MODULE=auto
ENV SRCPATH $GOPATH/src/github.com/form3tech-oss/$APPNAME

COPY ./ $SRCPATH

RUN go install github.com/form3tech-oss/$APPNAME/cmd/$APPNAME

FROM alpine

ARG APPNAME
ARG TAGS
ENV SERVICE_TAGS=$TAGS,active
ENV LOG_FORMAT=json
ENV AWS_SDK_LOAD_CONFIG=1
RUN apk add --no-cache dumb-init ca-certificates
WORKDIR /app
COPY build/package/$APPNAME/entrypoint.sh /app/
COPY --from=build-env /go/bin/$APPNAME /app/

ENTRYPOINT ./entrypoint.sh
