ARG GOLANG_VERSION="1"

FROM golang:${GOLANG_VERSION} AS build_zjddns

WORKDIR /zjddns

ADD . /zjddns

ENV \
    CGO_ENABLED="0"

RUN \
    wget "https://curl.se/ca/cacert.pem" \
    && BUILD_TIME=$(date -u '+%Y-%m-%d_%H:%M:%S_UTC') \
    && COMMIT_SHA=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown") \
    && go build -o zjddns -trimpath -ldflags "-s -w -buildid= -X main.BuildTime=${BUILD_TIME} -X main.CommitHash=${COMMIT_SHA}" ./cmd/zjddns

FROM scratch AS rebase_zjddns

COPY --from=build_zjddns /zjddns/cacert.pem /etc/ssl/certs/ca-certificates.crt
COPY --from=build_zjddns /zjddns/zjddns /zjddns

FROM scratch

COPY --from=rebase_zjddns / /

ENTRYPOINT ["/zjddns"]
