#!/bin/sh

if [ ! -f "epel-release-7-8.noarch.rpm" ]; then
  wget http://dl.fedoraproject.org/pub/epel/7/x86_64/e/epel-release-7-8.noarch.rpm
fi

mkdir -p _output
cp Dockerfile _output
cp *.py _output
cp *.rpm _output
sudo docker build -t example-prometheus:latest _output
rm -rf _output
