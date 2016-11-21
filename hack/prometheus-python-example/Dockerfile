FROM centos:centos7

EXPOSE 8181

ADD epel-release-7-8.noarch.rpm /opt
RUN rpm -iUvh /opt/epel-release-7-8.noarch.rpm && \
    yum install -y -q go git wget make python-pip && \
    yum clean all && \
    pip install prometheus_client

RUN mkdir -p /opt/example

COPY prometheus-python.py /opt/example

ENTRYPOINT ["/opt/example/prometheus-python.py"]
