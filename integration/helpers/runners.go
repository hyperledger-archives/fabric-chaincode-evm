/*
Copyright IBM Corp All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package helpers

import (
	"fmt"
	"os/exec"
	"time"

	"github.com/tedsuo/ifrit/ginkgomon"
)

func FabProxyRunner(fabproxyBinPath, proxyConfig, org, user, channel, ccid string, port uint16) *ginkgomon.Runner {
	cmd := exec.Command(fabproxyBinPath)
	cmd.Env = append(cmd.Env, fmt.Sprintf("FABPROXY_CONFIG=%s", proxyConfig))
	cmd.Env = append(cmd.Env, fmt.Sprintf("FABPROXY_ORG=%s", org))
	cmd.Env = append(cmd.Env, fmt.Sprintf("FABPROXY_USER=%s", user))
	cmd.Env = append(cmd.Env, fmt.Sprintf("FABPROXY_CHANNEL=%s", channel))
	cmd.Env = append(cmd.Env, fmt.Sprintf("FABPROXY_CCID=%s", ccid))
	cmd.Env = append(cmd.Env, fmt.Sprintf("PORT=%d", port))

	config := ginkgomon.Config{
		Name:              fmt.Sprintf("fabproxy-%s-%s", org, user),
		Command:           cmd,
		StartCheck:        "Starting Fab3 on port",
		StartCheckTimeout: 15 * time.Second,
	}

	return ginkgomon.New(config)
}

func Web3TestRunner(proxyAddr1, proxyAddr2 string) *ginkgomon.Runner {
	cmd := exec.Command("node", "web3_e2e_test.js", proxyAddr1, proxyAddr2)

	config := ginkgomon.Config{
		Name:              "web3-e2e",
		Command:           cmd,
		StartCheck:        "Starting Web3 E2E Test",
		StartCheckTimeout: 15 * time.Second,
	}
	return ginkgomon.New(config)
}
