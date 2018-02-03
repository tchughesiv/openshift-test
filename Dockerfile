FROM centos:centos7
RUN yum -y install iptables container-storage-setup && \
    yum clean all
ARG b
COPY ${b} /usr/bin/sccoc
RUN chmod +x /usr/bin/sccoc
ENTRYPOINT [ "sccoc" ]
CMD [ "run", "-h" ]
