FROM golang:1.25-alpine AS checkerbuild

WORKDIR /root

COPY checker /root/checker
WORKDIR /root/checker
RUN go test .
RUN go build -o chart-checker .

FROM alpine:3.22

RUN apk add --no-cache curl bash

# Install python3 and pip
RUN apk add --no-cache python3 py3-pip openssl git

RUN curl -fsSL https://get.helm.sh/helm-v3.19.0-linux-amd64.tar.gz -o helm.tar.gz && \
    tar -zxvf helm.tar.gz && \
    mv linux-amd64/helm /usr/local/bin/helm && \
    chmod +x /usr/local/bin/helm && \
    rm helm.tar.gz

# Install helm unittest
RUN helm plugin install https://github.com/helm-unittest/helm-unittest.git

# Install kustomize
RUN curl -s "https://raw.githubusercontent.com/kubernetes-sigs/kustomize/master/hack/install_kustomize.sh" -o install_kustomize.sh

# Install kubeconform form alpine
RUN curl -sSL "https://github.com/yannh/kubeconform/releases/download/v0.7.0/kubeconform-linux-amd64.tar.gz" -o /root/kubeconform.tar.gz
RUN tar -xzf /root/kubeconform.tar.gz -C /usr/local/bin kubeconform
RUN chmod +x /usr/local/bin/kubeconform

RUN chmod +x install_kustomize.sh
RUN ./install_kustomize.sh /usr/local/bin

# Make should also be available
RUN apk add --no-cache make

# Install docker client
RUN apk add --no-cache docker-cli

# Install docker-credential-gcr for Google Artifact Registry authentication
# This is much lighter (~20MB) than full gcloud SDK (~1GB+)
RUN curl -fsSL "https://github.com/GoogleCloudPlatform/docker-credential-gcr/releases/download/v2.1.26/docker-credential-gcr_linux_amd64-2.1.26.tar.gz" | tar xz -C /usr/local/bin docker-credential-gcr \
    && chmod +x /usr/local/bin/docker-credential-gcr

# Configure docker to use the GCR credential helper
RUN mkdir -p /root/.docker \
    && echo '{"credHelpers":{"gcr.io":"gcr","us-docker.pkg.dev":"gcr","europe-docker.pkg.dev":"gcr","asia-docker.pkg.dev":"gcr"}}' > /root/.docker/config.json

# Copy chart-checker from the builder stage
COPY --from=checkerbuild /root/checker/chart-checker /usr/local/bin/chart-checker

WORKDIR /app