FROM alpine:3.7
LABEL maintainer="Christopher Aaron <chriswainwrightaaron@gmail.com>"
COPY incommonca.crt /root/incommonca.crt
COPY amoss /usr/bin/amoss
#COPY prod/ /root/prod
COPY dev/ /root/dev
WORKDIR /root
EXPOSE 8080
ENTRYPOINT ["amoss", "-dev"]
#ENTRYPOINT ["amoss", "-prod"]
## Entrypoint Flags:
## Application: "amoss"
## Environment: "-prod", "-prod"
