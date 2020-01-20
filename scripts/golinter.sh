#!/bin/bash -e

# Copyright Greg Haskins All Rights Reserved.
#
# SPDX-License-Identifier: Apache-2.0

declare -a vendoredModules=(
"./evmcc/evmcc*.go"
"./evmcc/eventmanager"
"./evmcc/event"
"./evmcc/statemanager"
"./evmcc/address"
"./integration/e2e"
"./integration/fab3"
"./integration/helpers"
)

declare -a goModules=(
"./fab3"
)

for i in "${vendoredModules[@]}"
do
    echo ">>>Checking $i with goimports"
    OUTPUT="$(goimports -l ./$i || true )"
    if [[ $OUTPUT ]]; then
        echo "The following files contain goimports errors"
        echo $OUTPUT
        echo "The goimports command 'goimports -l -w' must be run for these files"
        exit 1
    fi
done

for i in "${goModules[@]}"
do
    echo ">>>Checking $i with goimports"
    OUTPUT="$(goimports -l ./$i || true )"
    if [[ $OUTPUT ]]; then
        echo "The following files contain goimports errors"
        echo $OUTPUT
        echo "The goimports command 'goimports -l -w' must be run for these files"
        exit 1
    fi
done

for i in "${vendoredModules[@]}"
do
    echo ">>>Checking $i with go vet"
    OUTPUT="$(GO111MODULE=off go vet ./$i)"
    if [[ $OUTPUT ]]; then
        echo "The following files contain go vet errors"
        echo $OUTPUT
        exit 1
    fi
done

for i in "${goModules[@]}"
do
    echo ">>>Checking $i with go vet"
    pushd $i
        OUTPUT="$(GO111MODULE=on go vet .)"
        exitCode=$?
    popd

    if [[ exitCode -eq 1 ]]; then
        echo "The following files contain go vet errors"
        echo $OUTPUT
        exit 1
    fi
done
