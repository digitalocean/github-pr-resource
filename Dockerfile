FROM golang:1.11 as builder
ADD . /go/src/github.com/telia-oss/github-pr-resource
WORKDIR /go/src/github.com/telia-oss/github-pr-resource
ENV TARGET=linux ARCH=amd64
RUN make build

FROM alpine:3.8 as resource
COPY --from=builder /go/src/github.com/telia-oss/github-pr-resource/build /opt/resource
RUN apk add --update --no-cache \ 
    git \
    openssh \
    && chmod +x /opt/resource/*

FROM resource
LABEL MAINTAINER=telia-oss
