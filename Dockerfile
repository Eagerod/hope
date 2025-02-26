FROM golang:1.23 AS builder

WORKDIR /app

RUN \
    apt-get update && \
    apt-get install -y \
        gettext-base && \
    apt-get clean

RUN go install honnef.co/go/tools/cmd/staticcheck@v0.6.0

COPY go.mod go.sum ./

RUN go mod download

ARG VERSION UnspecifiedContainerVersion

COPY . .

RUN \
  make test && \
  make && \
  gofmt -l . | grep . && echo "go fmt wants to make changes; run go fmt and fix linting errors." && exit 1 || exit 0


FROM debian:12

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

# Docker
RUN \
    install -m 0755 -d /etc/apt/keyrings && \
    curl -fsSL https://download.docker.com/linux/debian/gpg -o /etc/apt/keyrings/docker.asc && \
    chmod a+r /etc/apt/keyrings/docker.asc && \
    echo \
        "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.asc] https://download.docker.com/linux/debian \
        $(. /etc/os-release && echo "$VERSION_CODENAME") stable" > /etc/apt/sources.list.d/docker.list && \
    apt-get update && \
    apt-get install -y \
        docker-ce \
        docker-ce-cli \
        containerd.io && \
    apt-get clean

# Packer
RUN \
    curl -fsS https://apt.releases.hashicorp.com/gpg | apt-key add - && \
    apt-add-repository "deb [arch=amd64] https://apt.releases.hashicorp.com $(lsb_release -cs) main" && \
    apt-get update && \
    apt-get install -y packer && \
    apt-get clean

# Kubectl
ARG KUBERNETES_VERSION=v1.32.1

RUN \
    ARCH="amd64" && \
    curl -fsSL "https://dl.k8s.io/release/${KUBERNETES_VERSION}/bin/linux/${ARCH}/kubectl" -o /usr/bin/kubectl && \
    curl -fsSL "https://dl.k8s.io/release/${KUBERNETES_VERSION}/bin/linux/${ARCH}/kubeadm" -o /usr/bin/kubeadm && \
    curl -fsSL "https://dl.k8s.io/release/${KUBERNETES_VERSION}/bin/linux/${ARCH}/kubelet" -o /usr/bin/kubelet && \
    chmod +x /usr/bin/kubeadm /usr/bin/kubelet /usr/bin/kubectl

COPY --from=builder /app/build/hope /usr/bin/hope

VOLUME ["/src"]
