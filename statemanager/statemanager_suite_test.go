/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package statemanager_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestStatemanager(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Statemanager Suite")
}
