#!/usr/bin/env bash

CLUSTER_NAME="harbor-operator-integration-tests"

source ./teardown.sh

function checkDependencies() {
    echo "Checking for existence of necessary tools..."

    kind version &>/dev/null
    if [[ $? -ne "0" ]]; then
        >&2 echo "kind is not installed, aborting."
        exit 1
    fi

    docker --version &>/dev/null
    if [[ $? -ne "0" ]]; then
        >&2 echo "Docker is not installed, aborting."
        exit 1
    fi


    helm_version="$(helm version --short)"
    if ! [[ ${helm_version} =~ ^v3. ]]; then
        >&2 echo "Helm is not installed or not v3, aborting."
        exit 1
    fi

    kubectl_version="$(kubectl version --short)"
    if [[ -z ${kubectl_version} ]]; then
        >&2 echo "Kubectl is not installed, aborting."
        exit 1
    fi
}

function createCluster() {
    kind delete cluster --name "${CLUSTER_NAME}"

    kind create cluster --config test/kind-config.yaml --name "${CLUSTER_NAME}"
    if [[ "$?" -ne "0" ]]; then
        >&2 echo "Could not create kind cluster, aborting."
        exit 1
    fi

    export KUBECONFIG=/tmp/"${CLUSTER_NAME}".kubeconfig

    echo "Saving temporary kubeconfig for kind cluster to ${KUBECONFIG}"
    kind get kubeconfig --name "${CLUSTER_NAME}" > "${KUBECONFIG}"
    if [[ "$?" -ne "0" ]]; then
        >&2 echo "Could not create temporary kubeconfig for kind cluster, aborting."
        exit 1
    fi

    echo "Creating harbor-operator namespace"
    $KUBECONFIG; kubectl create namespace harbor-operator
    if [[ "$?" -ne "0" ]]; then
        >&2 echo "Could not create harbor-operator namespace, aborting."
        exit 1
    fi

    chmod 0644 "${KUBECONFIG}"
    if [[ "$?" -ne "0" ]]; then
        >&2 echo "Could not set permissions on temporary kubeconfig, aborting."
        exit 1
    fi

    echo "Installing CRDs ..."
        $KUBECONFIG; kubectl create -f deploy/crds/
    if [[ "$?" -ne "0" ]]; then
        >&2 echo "Could not install CRDs, aborting."
        exit 1
    fi

    echo "Successfully installed all harbor-operator dependencies."
}

function err() {
    cleanup
    exit 1
}

checkDependencies || err
createCluster || err
