/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package fabproxy_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestFabproxy(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Fabproxy Main Suite")
}
