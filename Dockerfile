FROM registry.access.redhat.com/ubi9-minimal:9.5
COPY out/cert-mgr-http01-proxy /usr/bin/
EXPOSE 6666
EXPOSE 7777
EXPOSE 8888
EXPOSE 9999
USER 9999
CMD ["/usr/bin/cert-mgr-http01-proxy"]