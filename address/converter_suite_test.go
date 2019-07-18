/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package address_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestAddressgenerator(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Addressgenerator Suite")
}
