# Current Version: 1.0.5

FROM alpine:latest AS build_baseos

COPY ./CloudflareDDNS.sh /opt/CloudflareDDNS.sh

RUN sed -i "s/dl-cdn.alpinelinux.org/mirrors.ustc.edu.cn/g" "/etc/apk/repositories" \
    && apk update \
    && apk add --no-cache bind-tools curl jq \
    && apk upgrade --no-cache \
    && curl -s --connect-timeout 15 "https://curl.se/ca/cacert.pem" > "/etc/ssl/certs/cacert.pem" && mv "/etc/ssl/certs/cacert.pem" "/etc/ssl/certs/ca-certificates.crt" \
    && rm -rf /tmp/* /var/cache/apk/*

FROM scratch

COPY --from=build_baseos / /

ENV XAUTHEMAIL=${XAUTHEMAIL} XAUTHKEY=${XAUTHKEY} ZONENAME=${ZONENAME} RECORDNAME=${RECORDNAME} TYPE=${TYPE} TTL=${TTL} STATICIP=${STATICIP} PROXYSTATUS=${PROXYSTATUS} RUNNINGMODE=${RUNNINGMODE} UPDATEFREQUENCY=${UPDATEFREQUENCY}

CMD [ "/bin/sh", "-c", "sh '/opt/CloudflareDDNS.sh' -e ${XAUTHEMAIL:-demo@zhijie.online} -k ${XAUTHKEY:-123defghijk4567pqrstuvw890} -z ${ZONENAME:-zhijie.online} -r ${RECORDNAME:-demo.zhijie.online} -t ${TYPE:-A} -l ${TTL:-3600} -i ${STATICIP:-auto} -p ${PROXYSTATUS:-false} -m ${RUNNINGMODE:-update} && sleep ${UPDATEFREQUENCY:-3600}" ]
