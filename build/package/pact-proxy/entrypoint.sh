#!/usr/bin/dumb-init /bin/sh
set -e
ip -4 route list match 0/0 | awk '{print $3 "host.docker.internal"}' >> /etc/hosts
./pact-proxy
