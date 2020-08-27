#!/usr/bin/env bash

function cleanup() {
    kind version &>/dev/null
    if [[ $? -ne "0" ]]; then
        >&2 echo "kind not installed, aborting."
        exit 1
    fi

    echo "Deleting existing kind cluster..."
    kind delete cluster --name "${CLUSTER_NAME}"
    if [[ "$?" -ne "0" ]]; then
        >&2 echo "Could not delete kind cluster, aborting."
        exit 1
    fi

    echo "Deleting temporary kubeconfig ${KUBECONFIG}"
    if [ -f "${KUBECONFIG}" ];then
        rm "${KUBECONFIG}"
    fi
    if [[ "$?" -ne "0" ]]; then
        >&2 echo "Could not delete temporary kubeconfig ${KUBECONFIG}"
        exit 1
    fi

    echo "Successfully uninstalled all harbor-operator dependencies."
}