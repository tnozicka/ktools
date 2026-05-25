FROM quay.io/tnozicka/images:golang-1.26 as builder
WORKDIR /root/go/src/github.com/tnozicka/ktools/
COPY . .
RUN make build --warn-undefined-variables

FROM quay.io/tnozicka/images:base-fedora-44
LABEL org.opencontainers.image.title="KTool" \
      org.opencontainers.image.description="A collection of helpful Kubernetes tools" \
      org.opencontainers.image.authors="Tomas Nozicka" \
      org.opencontainers.image.source="https://github.com/tnozicka/ktools" \
      org.opencontainers.image.documentation="" \
      org.opencontainers.image.url="" \
      org.opencontainers.image.vendor="tnozicka"
COPY --from=builder /root/go/src/github.com/tnozicka/ktools/ktool /usr/local/bin/ktool
ENTRYPOINT ["/usr/local/bin/ktool"]
