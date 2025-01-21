FROM registry.access.redhat.com/ubi9-minimal:9.5
COPY out/cert-mgr-http01-proxy /usr/bin/
RUN setcap 'cap_net_bind_service+ep' /usr/bin/cert-mgr-http01-proxy
EXPOSE 80
USER 9999
CMD ["/usr/bin/cert-mgr-http01-proxy"]