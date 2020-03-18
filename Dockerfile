ARG ARCH="amd64"
ARG OS="linux"
FROM quay.io/prometheus/busybox-${OS}-${ARCH}:glibc
ARG ARCH="amd64"
ARG OS="linux"
COPY .build/${OS}-${ARCH}/lsdyna_exporter /lsdyna_exporter
EXPOSE 9309
USER nobody
ENTRYPOINT ["/lsdyna_exporter"]
