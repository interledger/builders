FROM golang:1.25-alpine

RUN apk add --no-cache curl bash

# Install python3 and pip
RUN apk add --no-cache python3 py3-pip openssl

# Install Helm
RUN curl -fsSL -o get_helm.sh https://raw.githubusercontent.com/helm/helm/master/scripts/get-helm-3
RUN chmod 700 get_helm.sh
RUN ./get_helm.sh

# Install kustomize
RUN curl -s "https://raw.githubusercontent.com/kubernetes-sigs/kustomize/master/hack/install_kustomize.sh" -o install_kustomize.sh

RUN chmod +x install_kustomize.sh
RUN ./install_kustomize.sh /usr/local/bin

# As described https://cloud.google.com/sdk/docs/install
RUN curl -O https://dl.google.com/dl/cloudsdk/channels/rapid/downloads/google-cloud-cli-linux-x86_64.tar.gz
RUN tar -xvf google-cloud-cli-linux-x86_64.tar.gz
RUN ./google-cloud-sdk/install.sh --quiet
RUN ./google-cloud-sdk/bin/gcloud components install kubectl

# Make should also be available
RUN apk add --no-cache make

WORKDIR /app