/*
Copyright IBM Corp. All Rights Reserved.
SPDX-License-Identifier: Apache-2.0
*/

package eventmanager_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestEventManager(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "EventManager Suite")
}
