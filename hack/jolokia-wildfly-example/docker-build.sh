#!/bin/sh

if [ ! -f "jolokia-war-1.3.5.war" ]; then
  wget http://repo1.maven.org/maven2/org/jolokia/jolokia-war/1.3.5/jolokia-war-1.3.5.war
fi

if [ ! -f "wildfly-10.1.0.Final.tar.gz" ]; then
  wget http://download.jboss.org/wildfly/10.1.0.Final/wildfly-10.1.0.Final.tar.gz
fi

mkdir -p _output
cp Dockerfile _output
cp wildfly-*.tar.gz _output
cp jolokia-war-*.war _output
sudo docker build -t jolokia-wildfly:latest _output
rm -rf _output
