FROM java:8

EXPOSE 8080

ADD wildfly-10.1.0.Final.tar.gz /opt
COPY jolokia-war-1.3.5.war /opt/wildfly-10.1.0.Final/standalone/deployments

ENV JBOSS_HOME /opt/wildfly-10.1.0.Final
RUN chgrp -R 0 $JBOSS_HOME
RUN chmod -R g+rw $JBOSS_HOME
RUN find $JBOSS_HOME -type d -exec chmod g+x {} +

CMD /opt/wildfly-10.1.0.Final/bin/standalone.sh -b 0.0.0.0 -bmanagement=0.0.0.0
