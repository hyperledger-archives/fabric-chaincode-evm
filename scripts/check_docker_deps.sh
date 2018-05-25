#!/bin/bash
#
# Copyright IBM Corp All Rights Reserved.
# SPDX-License-Identifier: Apache-2.0
#
docker inspect hyperledger/fabric-buildenv:latest &>/dev/null
if [ '$?' == '1' ]; then
  # TODO: use below once fabric-buildenv images published
  # docker pull hyperledger/fabric-buildenv:latest
  echo "ERROR: hyperledger/fabric-buildenv:latest image required" && exit 1
fi
docker inspect hyperledger/fabric-peer:latest &>/dev/null
if [ '$?' == '1' ]; then
  # TODO: use below once fabric 1.2 images published
  # docker pull hyperledger/fabric-peer:latest
  echo "ERROR: hyperledger/fabric-peer:latest image required" && exit 1
fi
