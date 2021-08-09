FROM alpine:3.6
MAINTAINER Christopher Aaron <chriswainwrightaaron@gmail.com>
ENV VAULT_ADDR=https://amossvault.emory.edu:8200
ENV VAULT_TOKEN=8c410835-cc39-d713-d9b6-dedcc68920b7
COPY incommonca.crt /root/incommonca.crt
COPY amoss_jawbone /root/amoss_jawbone
WORKDIR /root
ENTRYPOINT ["/root/amoss_jawbone","-prod"]