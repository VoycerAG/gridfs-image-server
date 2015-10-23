FROM debian:wheezy

ENV NEWRELIC_LICENSE ""
ENV MONGODB_CONNECTION "localhost:27017"



EXPOSE 8000

COPY configuration.json /etc/image-server/

VOLUME ["/etc/image-server"]

COPY build/image-server.x64.1.5 /image-server
COPY scripts/docker-wrapper /run-image-server

# we need this for stupid ssl certificates
# if we find a better solution for certs
# we can run this image from busybox:ubuntu-14.04
# and it will be a looot smaller
RUN apt-get update
RUN apt-get install -y ca-certificates 

CMD /bin/sh run-image-server
