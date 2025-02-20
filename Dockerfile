FROM registry.access.redhat.com/ubi9-minimal:9.5
COPY out/cert-mgr-http01-proxy /usr/bin/
# CAP_NET_ADMIN, CAP_DAC_READ_SEARCH and CAP_NET_RAW required to inject iptables rules
RUN setcap 'cap_net_admin,cap_dac_read_search,cap_net_raw+ep' /usr/bin/cert-mgr-http01-proxy
RUN microdnf install iptables -y && microdnf clean all
EXPOSE 6666
EXPOSE 7777
EXPOSE 8888
EXPOSE 9999
USER 9999
CMD ["/usr/bin/cert-mgr-http01-proxy"]