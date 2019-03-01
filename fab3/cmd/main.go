/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"go.uber.org/zap"

	"github.com/hyperledger/fabric-chaincode-evm/fab3"
	"github.com/hyperledger/fabric-sdk-go/pkg/client/channel"
	"github.com/hyperledger/fabric-sdk-go/pkg/client/ledger"
	"github.com/hyperledger/fabric-sdk-go/pkg/core/config"
	"github.com/hyperledger/fabric-sdk-go/pkg/fabsdk"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var fab3Cmd = &cobra.Command{
	Use:   "fab3",
	Short: "fab3 is a web3 provider used to interact with the EVM chaincode on a Fabric Network. The flags provided will be honored over the corresponding environment variables.",
	Long:  "fab3 is a web3 provider used to interact with the EVM chaincode on a Fabric Network. The flags provided will be honored over the corresponding environment variables.",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		err := checkFlags()
		if err != nil {
			return err
		}

		// At this point all of our flags have been validated
		// Usage no longer needs to be provided for the errors that follow
		cmd.SilenceUsage = true
		return runFab3(cmd, args)
	},
}

var cfg, user, org, ch, ccid string
var port int

// InitFlags sets up the flags and environment variables for Fab3
func initFlags() {
	viper.SetEnvPrefix("FAB3")
	viper.BindEnv("config")
	viper.BindEnv("user")
	viper.BindEnv("org")
	viper.BindEnv("channel")
	viper.BindEnv("ccid")
	viper.BindEnv("port")

	fab3Cmd.PersistentFlags().StringVarP(&cfg, "config", "c", "",
		"Path to a compatible Fabric SDK Go config file. This flag is required if FAB3_CONFIG is not set.")
	viper.BindPFlag("config", fab3Cmd.PersistentFlags().Lookup("config"))

	fab3Cmd.PersistentFlags().StringVarP(&user, "user", "u", "",
		"User identity being used for the proxy (Matches the users' names in the crypto-config directory specified in the config). This flag is required if FAB3_USER is not set.")
	viper.BindPFlag("user", fab3Cmd.PersistentFlags().Lookup("user"))

	fab3Cmd.PersistentFlags().StringVarP(&org, "org", "o", "",
		"Organization of the specified user. This flag is required if FAB3_ORG is not set.")
	viper.BindPFlag("org", fab3Cmd.PersistentFlags().Lookup("org"))

	fab3Cmd.PersistentFlags().StringVarP(&ch, "channel", "C", "",
		"Channel to be used for the transactions. This flag is required if FAB3_CHANNEL is not set.")
	viper.BindPFlag("channel", fab3Cmd.PersistentFlags().Lookup("channel"))

	//CCID defaults to "evmcc" if FAB3_CCID is not set or `-i,-ccid` is not provided
	fab3Cmd.PersistentFlags().StringVarP(&ccid, "ccid", "i", "evmcc",
		"ID of the EVM Chaincode deployed in your fabric network. The CCID to be used in by fab3 can also be set by the FAB3_CCID environment variable.")
	viper.BindPFlag("ccid", fab3Cmd.PersistentFlags().Lookup("ccid"))

	//Port defaults to 5000 if PORT is not set or `-p,-port` is not provided
	fab3Cmd.PersistentFlags().IntVarP(&port, "port", "p", 5000,
		"Port that Fab3 will be running on. The listening port can also be set by the FAB3_PORT environment variable.")
	viper.BindPFlag("port", fab3Cmd.PersistentFlags().Lookup("port"))
}

// Viper takes care of precedence of Flags and Environment variables
// Flag values are taken over environment variables
// Both CCID and Port have defaults so do not need to be provided.
func checkFlags() error {
	cfg = viper.GetString("config")
	if cfg == "" {
		return fmt.Errorf("Missing config. Please use flag --config or set FAB3_CONFIG")
	}

	user = viper.GetString("user")
	if user == "" {
		return fmt.Errorf("Missing user. Please use flag --user or set FAB3_USER")
	}

	org = viper.GetString("org")
	if org == "" {
		return fmt.Errorf("Missing org. Please use flag --org or set FAB3_ORG")
	}

	ch = viper.GetString("channel")
	if ch == "" {
		return fmt.Errorf("Missing channel. Please use flag --channel or set FAB3_CHANNEL")
	}

	ccid = viper.GetString("ccid")
	port = viper.GetInt("port")
	return nil
}

// Runs Fab3
// Will exit gracefully for errors and signal interrupts
func runFab3(cmd *cobra.Command, args []string) error {
	sdk, err := fabsdk.New(config.FromFile(cfg))
	if err != nil {
		return fmt.Errorf("Failed to create Fabric SDK Client: %s\n", err)
	}

	clientChannelContext := sdk.ChannelContext(ch, fabsdk.WithUser(user), fabsdk.WithOrg(org))
	client, err := channel.New(clientChannelContext)
	if err != nil {
		sdk.Close()
		return fmt.Errorf("Failed to create Fabric SDK Channel Client: %s\n", err)
	}

	ledger, err := ledger.New(clientChannelContext)
	if err != nil {
		sdk.Close()
		return fmt.Errorf("Failed to create Fabric SDK Ledger Client: %s\n", err)
	}

	rawLogger, err := zap.NewProduction()
	if err != nil {
		sdk.Close()
		return fmt.Errorf("Failed to create logger: %s\n", err)
	}
	logger := rawLogger.Named("fab3").Sugar()

	ethService := fab3.NewEthService(client, ledger, ch, ccid, logger)

	proxy := fab3.NewFab3(ethService)

	errChan := make(chan error, 1)
	go func() {
		errChan <- proxy.Start(port)
	}()
	logger.Infow("Starting Fab3", "port", port)

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM)

	select {
	case err = <-errChan:
	case <-signalChan:
		logger.Info("Received termination signal")
		err = proxy.Shutdown()
	}

	sdk.Close()
	if err != nil {
		logger.Infow("Fab3 exited with error", "error", err)
		return err
	}
	logger.Info("Fab3 exited")
	return nil
}

func main() {
	initFlags()
	if fab3Cmd.Execute() != nil {
		os.Exit(1)
	}
}
