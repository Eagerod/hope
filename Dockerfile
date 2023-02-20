FROM golang:1.19 AS builder

WORKDIR /app

RUN \
    apt-get update && \
    apt-get install -y \
        gettext-base && \
    apt-get clean

COPY go.mod go.sum ./

RUN go mod download

ARG VERSION UnspecifiedContainerVersion

COPY . .

RUN make test && make


FROM debian:10

RUN \
    apt-get update && \
    apt-get install -y \
        apache2-utils \
        apt-transport-https \
        build-essential \
        curl \
        gettext-base \
        gnupg2 \
        lsb-release \
        python3-pip \
        sshpass \
        software-properties-common && \
    apt-get clean

# Kubectl
RUN curl -fsS https://packages.cloud.google.com/apt/doc/apt-key.gpg | apt-key add -
RUN echo "deb https://apt.kubernetes.io/ kubernetes-xenial main" | tee -a /etc/apt/sources.list.d/kubernetes.list

# Packer
RUN curl -fsS https://apt.releases.hashicorp.com/gpg | apt-key add -
RUN apt-add-repository "deb [arch=amd64] https://apt.releases.hashicorp.com $(lsb_release -cs) main"

# Docker
RUN curl -fsS https://download.docker.com/linux/debian/gpg | apt-key add -
RUN add-apt-repository "deb [arch=amd64] https://download.docker.com/linux/debian $(lsb_release -cs) stable"

RUN \
    apt-get update && \
    apt-get install -y \
        containerd.io \
        docker-ce \
        docker-ce-cli \
        kubectl \
        packer && \
    apt-get clean

COPY --from=builder /app/build/hope /usr/bin/hope

VOLUME ["/src"]
