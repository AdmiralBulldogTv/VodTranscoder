FROM golang:1.17.6 as builder

WORKDIR /tmp/transcoder

COPY . .

ARG BUILDER
ARG VERSION

ENV TRANSCODER_BUILDER=${BUILDER}
ENV TRANSCODER_VERSION=${VERSION}

RUN apt-get update && apt-get install make git gcc -y && \
    make build_deps && \
    make

FROM alfg/ffmpeg:latest

WORKDIR /app

COPY --from=builder /tmp/transcoder/bin/transcoder .

CMD ["/app/transcoder"]
