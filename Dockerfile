FROM golang:1.11 as builder
MAINTAINER telia-oss
ADD . /go/src/github.com/telia-oss/github-pr-resource
WORKDIR /go/src/github.com/telia-oss/github-pr-resource
ENV TARGET linux
ENV ARCH amd64
RUN make build

FROM alpine:3.8 as resource
RUN apk --update add git openssh
COPY --from=builder /go/src/github.com/telia-oss/github-pr-resource/check /opt/resource/check
COPY --from=builder /go/src/github.com/telia-oss/github-pr-resource/in /opt/resource/in
COPY --from=builder /go/src/github.com/telia-oss/github-pr-resource/out /opt/resource/out
RUN chmod +x /opt/resource/*

FROM resource
