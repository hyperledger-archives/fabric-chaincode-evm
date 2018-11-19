/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"os"
	"strconv"

	"go.uber.org/zap"

	"github.com/hyperledger/fabric-chaincode-evm/fabproxy"
	"github.com/hyperledger/fabric-sdk-go/pkg/client/channel"
	"github.com/hyperledger/fabric-sdk-go/pkg/client/ledger"
	"github.com/hyperledger/fabric-sdk-go/pkg/core/config"
	"github.com/hyperledger/fabric-sdk-go/pkg/fabsdk"
)

const usage = `FabProxy uses environment variables to be able to start communicating with a Fabric network
	Required Environment Variables:
	  FABPROXY_CONFIG - Path to a compatible Fabric SDK Go config file
	  FABPROXY_USER - User identity being used for the proxy (Matches the users names in the crypto-config directory specified in the config)
	  FABPROXY_ORG - Organization of the specified user
	  FABPROXY_CHANNEL - Channel to be used for the transactions
	  FABPROXY_CCID - ID of the EVM Chaincode deployed in your fabric network

	Other Environment Variables:
	  PORT - Port the FabProxy will be running on. Default is 5000
	`

var logger *zap.SugaredLogger

func main() {
	rawLogger, _ := zap.NewProduction()
	logger := rawLogger.Named("fab3").Sugar()

	cfg := grabEnvVar("FABPROXY_CONFIG", true, "Path to the Fabric SDK GO config file")
	org := grabEnvVar("FABPROXY_ORG", true, "Org of the user specified")
	user := grabEnvVar("FABPROXY_USER", true, "User to be used for proxy")
	ch := grabEnvVar("FABPROXY_CHANNEL", true, "Channel transactions will be sent on")
	ccid := grabEnvVar("FABPROXY_CCID", true, "Chaincode ID of the EVM chaincode")
	port := grabEnvVar("PORT", false, "")

	portNumber := 5000
	if port != "" {
		var err error
		portNumber, err = strconv.Atoi(port)
		if err != nil {
			logger.Fatalf("Failed to convert the environment variable `PORT`, %s,  to an int\n", port)
		}
	}

	sdk, err := fabsdk.New(config.FromFile(cfg))
	if err != nil {
		logger.Fatalf("Failed to create Fabric SDK Client: %s\n", err.Error())
	}
	defer sdk.Close()

	clientChannelContext := sdk.ChannelContext(ch, fabsdk.WithUser(user), fabsdk.WithOrg(org))
	client, err := channel.New(clientChannelContext)
	if err != nil {
		logger.Fatalf("Failed to create Fabric SDK Channel Client: %s\n", err.Error())
	}

	ledger, err := ledger.New(clientChannelContext)
	if err != nil {
		logger.Fatalf("Failed to create Fabric SDK Ledger Client: %s\n", err.Error())
	}

	ethService := fabproxy.NewEthService(client, ledger, ch, ccid, logger)

	logger.Infof("Starting Fab Proxy on port %d\n", portNumber)
	proxy := fabproxy.NewFabProxy(ethService)
	err = proxy.Start(portNumber)
	if err != nil {
		logger.Fatal(err)
	}
	defer func() {
		logger.Info("Shutting down the Fab Proxy")
		proxy.Shutdown()
		logger.Info("Fab Proxy has exited")
	}()
}

func grabEnvVar(varName string, required bool, description string) string {
	envVar := os.Getenv(varName)
	if required && envVar == "" {
		logger.Fatalf("Fab Proxy requires the environment variable %s to be set\n\n%s\n\n", varName, usage)
	}
	return envVar
}
