/*
Copyright IBM Corp All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package fab3

import (
	"fmt"
	"os/exec"
	"strconv"

	"github.com/hyperledger/fabric-chaincode-evm/integration/helpers"
	"github.com/onsi/ginkgo/config"
	"github.com/onsi/gomega/gbytes"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Fab3 Configuration", func() {
	var (
		proxyCmd  *exec.Cmd
		proxyPort uint16
	)

	BeforeEach(func() {
		proxyPort = uint16(6000 + config.GinkgoConfig.ParallelNode)
	})

	AfterEach(func() {
		if proxyCmd != nil && proxyCmd.Process != nil {
			proxyCmd.Process.Kill()
		}
	})

	It("can be configured with environment variables", func() {
		proxyCmd = exec.Command(components.Paths["fab3"])
		proxyCmd.Env = append(proxyCmd.Env, fmt.Sprintf("FAB3_CONFIG=%s", components.Paths["Fab3Config"]))
		proxyCmd.Env = append(proxyCmd.Env, "FAB3_ORG=Org1")
		proxyCmd.Env = append(proxyCmd.Env, "FAB3_USER=User1")
		proxyCmd.Env = append(proxyCmd.Env, "FAB3_CHANNEL=testchannel")
		proxyCmd.Env = append(proxyCmd.Env, "FAB3_CCID=evmcc")
		proxyCmd.Env = append(proxyCmd.Env, fmt.Sprintf("FAB3_PORT=%d", proxyPort))

		output := gbytes.NewBuffer()
		proxyCmd.Stdout = output
		proxyCmd.Stderr = output

		err := proxyCmd.Start()
		Expect(err).ToNot(HaveOccurred())
		helpers.WaitForFab3(proxyPort)
	})

	It("can be configured with flags", func() {
		proxyCmd = exec.Command(components.Paths["fab3"],
			"--config", components.Paths["Fab3Config"],
			"--org", "Org1",
			"--user", "User1",
			"--channel", "testchannel",
			"--ccid", "evmcc",
			"--port", strconv.FormatUint(uint64(proxyPort), 10),
		)

		output := gbytes.NewBuffer()
		proxyCmd.Stdout = output
		proxyCmd.Stderr = output

		err := proxyCmd.Start()
		Expect(err).ToNot(HaveOccurred())
		helpers.WaitForFab3(proxyPort)
	})

	It("will use flag values over environment variables ", func() {
		proxyCmd = exec.Command(components.Paths["fab3"],
			"--config", components.Paths["Fab3Config"],
			"--org", "Org1",
			"--user", "User1",
			"--channel", "testchannel",
			"--ccid", "evmcc",
			"--port", strconv.FormatUint(uint64(proxyPort), 10),
		)
		proxyCmd.Env = append(proxyCmd.Env, "FAB3_CONFIG=non-existent-config-path")
		proxyCmd.Env = append(proxyCmd.Env, "FAB3_ORG=non-existent-org")
		proxyCmd.Env = append(proxyCmd.Env, "FAB3_USER=non-existent-user")
		proxyCmd.Env = append(proxyCmd.Env, "FAB3_CHANNEL=non-existent-channel")
		proxyCmd.Env = append(proxyCmd.Env, "FAB3_CCID=non-existent-ccid")
		proxyCmd.Env = append(proxyCmd.Env, "FAB3_PORT=fake-port")

		output := gbytes.NewBuffer()
		proxyCmd.Stdout = output
		proxyCmd.Stderr = output

		err := proxyCmd.Start()
		Expect(err).ToNot(HaveOccurred())
		helpers.WaitForFab3(proxyPort)
	})

	It("requires config to be set", func() {
		proxyCmd = exec.Command(components.Paths["fab3"],
			"--org", "Org1",
			"--user", "User1",
			"--channel", "testchannel",
			"--ccid", "evmcc",
			"--port", strconv.FormatUint(uint64(proxyPort), 10),
		)
		proxyCmd.Env = []string{""}

		output := gbytes.NewBuffer()
		proxyCmd.Stdout = output
		proxyCmd.Stderr = output

		err := proxyCmd.Start()
		Expect(err).ToNot(HaveOccurred())

		exitErr := proxyCmd.Wait()
		Expect(exitErr).To(HaveOccurred())
		Eventually(output).Should(gbytes.Say("Missing config"))
	})

	It("requires org to be set", func() {
		proxyCmd = exec.Command(components.Paths["fab3"],
			"--config", components.Paths["Fab3Config"],
			"--user", "User1",
			"--channel", "testchannel",
			"--ccid", "evmcc",
			"--port", strconv.FormatUint(uint64(proxyPort), 10),
		)
		proxyCmd.Env = []string{""}

		output := gbytes.NewBuffer()
		proxyCmd.Stdout = output
		proxyCmd.Stderr = output

		err := proxyCmd.Start()
		Expect(err).ToNot(HaveOccurred())

		exitErr := proxyCmd.Wait()
		Expect(exitErr).To(HaveOccurred())
		Eventually(output).Should(gbytes.Say("Missing org"))
	})

	It("requires user to be set", func() {
		proxyCmd = exec.Command(components.Paths["fab3"],
			"--config", components.Paths["Fab3Config"],
			"--org", "Org1",
			"--channel", "testchannel",
			"--ccid", "evmcc",
			"--port", strconv.FormatUint(uint64(proxyPort), 10),
		)
		proxyCmd.Env = []string{""}

		output := gbytes.NewBuffer()
		proxyCmd.Stdout = output
		proxyCmd.Stderr = output

		err := proxyCmd.Start()
		Expect(err).ToNot(HaveOccurred())

		exitErr := proxyCmd.Wait()
		Expect(exitErr).To(HaveOccurred())
		Eventually(output).Should(gbytes.Say("Missing user"))
	})

	It("requires channel to be set", func() {
		proxyCmd = exec.Command(components.Paths["fab3"],
			"--config", components.Paths["Fab3Config"],
			"--org", "Org1",
			"--user", "User1",
			"--ccid", "evmcc",
			"--port", strconv.FormatUint(uint64(proxyPort), 10),
		)
		proxyCmd.Env = []string{""}

		output := gbytes.NewBuffer()
		proxyCmd.Stdout = output
		proxyCmd.Stderr = output

		err := proxyCmd.Start()
		Expect(err).ToNot(HaveOccurred())

		exitErr := proxyCmd.Wait()
		Expect(exitErr).To(HaveOccurred())
		Eventually(output).Should(gbytes.Say("Missing channel"))
	})

	It("does not require ccid or port because they have defaults", func() {
		proxyCmd = exec.Command(components.Paths["fab3"],
			"--config", components.Paths["Fab3Config"],
			"--org", "Org1",
			"--user", "User1",
			"--channel", "testchannel",
		)
		proxyCmd.Env = []string{""}

		output := gbytes.NewBuffer()
		proxyCmd.Stdout = output
		proxyCmd.Stderr = output

		err := proxyCmd.Start()
		Expect(err).ToNot(HaveOccurred())
		helpers.WaitForFab3(5000)
	})
})
