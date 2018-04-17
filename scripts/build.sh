#!/usr/bin/env bash
# Copyright IBM Corp. All Rights Reserved.
#
# SPDX-License-Identifier: Apache-2.0
#
set -e

FABRIC_PATH=$1
EVMSCC_PATH=$2

echo "installing govendor"
go get -u github.com/kardianos/govendor

cd $EVMSCC_PATH

echo "removing packages already vendored in fabric from evmscc"
comm -12 <(cd $EVMSCC_PATH && govendor list +v +l | awk '{print $2}' | sort) <(cd $FABRIC_PATH && govendor list +v +l | awk '{print $2}' | sort) | xargs -n1 -I{} rm -rf vendor/{}

echo "copying packages needed by evmscc to fabric vendor"
cp -r $EVMSCC_PATH/vendor/* $FABRIC_PATH/vendor/

echo "copying evmscc to fabric plugin"
cp -r $EVMSCC_PATH/plugin $FABRIC_PATH/
mkdir -p $FABRIC_PATH/vendor/github.com/hyperledger/fabric-chaincode-evm
cp -r $EVMSCC_PATH/statemanager $FABRIC_PATH/vendor/github.com/hyperledger/fabric-chaincode-evm/

echo "building evmscc plugin shared object to $LIB_DIR"
cd $FABRIC_PATH && go build -o $LIB_DIR/evmscc.so -buildmode=plugin ./plugin

