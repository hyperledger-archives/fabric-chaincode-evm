/*
Copyright NAVER Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package signing

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestSigning(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Signing Suite")
}
