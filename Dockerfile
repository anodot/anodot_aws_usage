FROM golang:1.13

ENV TERRAFORM_VERSION=1.0.4
RUN rm -rf /tmp/* && \
    rm -rf /var/cache/apk/* && \
    rm -rf /var/tmp/* && \
    apt update -y && apt install -y awscli curl jq python bash ca-certificates git openssl unzip zip wget && \
    cd /tmp && \
    wget https://releases.hashicorp.com/terraform/${TERRAFORM_VERSION}/terraform_${TERRAFORM_VERSION}_linux_amd64.zip && \
    unzip terraform_${TERRAFORM_VERSION}_linux_amd64.zip -d /usr/bin
   

WORKDIR /output
COPY . .
RUN  go get -v ./
