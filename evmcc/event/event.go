/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package event

type Event struct {
	Address string
	Data    string
	Topics  []string
}
