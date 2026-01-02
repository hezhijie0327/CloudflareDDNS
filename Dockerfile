ARG GOLANG_VERSION="1"

FROM golang:${GOLANG_VERSION} AS build_cloudflareddns

WORKDIR /cloudflareddns

ADD ./* /cloudflareddns

ENV \
    CGO_ENABLED="0"

RUN \
    wget "https://curl.se/ca/cacert.pem" \
    && BUILD_TIME=$(date -u '+%Y-%m-%d_%H:%M:%S_UTC') \
    && COMMIT_SHA=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown") \
    && go build -o cloudflareddns -trimpath -ldflags "-s -w -buildid= -X main.BuildTime=${BUILD_TIME} -X main.CommitHash=${COMMIT_SHA}"

FROM scratch AS rebase_cloudflareddns

COPY --from=build_cloudflareddns /cloudflareddns/cacert.pem /etc/ssl/certs/ca-certificates.crt
COPY --from=build_cloudflareddns /cloudflareddns/cloudflareddns /cloudflareddns

FROM scratch

COPY --from=rebase_cloudflareddns / /

ENTRYPOINT ["/cloudflareddns"]
