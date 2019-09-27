#!/bin/bash -e

# Copyright Greg Haskins All Rights Reserved.
#
# SPDX-License-Identifier: Apache-2.0

declare -a arr=(
"./event"
"./eventmanager"
"./evmcc"
"./fab3"
"./integration"
"./statemanager"
)

for i in "${arr[@]}"
do
    echo ">>>Checking code under $i/ with goimports"
    OUTPUT="$(goimports -l ./$i || true )"
    if [[ $OUTPUT ]]; then
        echo "The following files contain goimports errors"
        echo $OUTPUT
        echo "The goimports command 'goimports -l -w' must be run for these files"
        exit 1
    fi
done

echo "Checking with go vet"
OUTPUT="$(go vet ./...)"
if [[ $OUTPUT ]]; then
    echo "The following files contain go vet errors"
    echo $OUTPUT
    exit 1
fi
