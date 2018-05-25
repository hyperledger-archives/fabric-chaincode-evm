#!/bin/sh
# Copyright IBM Corp. All Rights Reserved.
# SPDX-License-Identifier: Apache-2.0
#
docker rmi $(docker images | grep '\<none\>' | awk '{print $3}') || true
