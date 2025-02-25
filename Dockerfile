FROM registry.access.redhat.com/ubi9-minimal:9.5
COPY out/cert-mgr-http01-proxy /usr/bin/
EXPOSE 8888
USER 9999
CMD ["/usr/bin/cert-mgr-http01-proxy"]