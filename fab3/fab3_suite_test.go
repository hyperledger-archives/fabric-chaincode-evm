/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package fab3_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestFab3(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Fab3 Main Suite")
}
