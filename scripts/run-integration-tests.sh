#!/bin/bash
#
# Copyright IBM Corp. All Rights Reserved.
#
# SPDX-License-Identifier: Apache-2.0

# Use ginkgo to run integration tests. If arguments are provided to the
# script, they are treated as the directories containing the tests to run.
# When no arguments are provided, all integration tests are executed.

set -e

fabric_chaincode_evm_dir="$(cd "$(dirname "$0")/.." && pwd)"
FABRIC_DIR=${GOPATH}/src/github.com/hyperledger/fabric

# find packages that contain "integration" in the import path
integration_dirs() {
    local packages="$1"

    go list -f {{.Dir}} "$packages" | grep -E '/integration($|/)' | sed "s,${fabric_chaincode_evm_dir},.,g"
}

main() {
    cd "$fabric_chaincode_evm_dir"

    local -a dirs=("$@")
    if [ "${#dirs[@]}" -eq 0 ]; then
        dirs=($(integration_dirs "./..."))
    fi

    type node >/dev/null 2>&1 || { echo >&2 "No node in PATH.  Aborting."; exit 1; }

    if ! npm ls --depth 1 web3@0.20.2 >/dev/null 2>&1 ; then
        echo "Installing web3@0.20.2"
        npm install web3@0.20.2
    fi

    #Check if Fabric is in the gopath. Fabric needs to be in the gopath for the integration tests
    if [ ! -d "${FABRIC_DIR}" ]; then
        echo "Downloading Fabric Branch v1.4.0"
        git clone https://github.com/hyperledger/fabric $FABRIC_DIR --branch v1.4.0 --single-branch --depth 1
    else
        FABRIC_VERSION=$(git -C ${FABRIC_DIR} describe)
        if [[ ${FABRIC_VERSION} != "v1.4.0" ]]; then
          echo "Please switch Fabric Repository to tag v1.4.0 before running these tests"
          echo "You can run in the Fabric Directory: git checkout v1.4.0"
          exit 1
        fi
    fi

    echo "Building CCENV image"
    pushd ${FABRIC_DIR}
        make ccenv
    popd

    echo "Running integration tests..."
    ginkgo -noColor -randomizeAllSpecs -race -keepGoing --slowSpecThreshold 80 -r "${dirs[@]}"
}

main "$@"
