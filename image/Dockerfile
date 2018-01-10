FROM centos:centos7
RUN yum -y install iptables && \
    yum clean all
ARG sccoc
COPY ${sccoc} /usr/bin/sccoc
RUN chmod +x /usr/bin/sccoc
ENTRYPOINT [ "sccoc" ]
CMD [ "run" ]