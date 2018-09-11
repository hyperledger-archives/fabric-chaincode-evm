/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package fabproxy

import "net/http"

const NetworkID = "fabric-evm"

// NetService returns data about the network the client is connected
// to.
type NetService struct {
}

// Version takes no parameters and returns the network identifier.
//
// https://github.com/ethereum/wiki/wiki/JSON-RPC#net_version
func (s *NetService) Version(r *http.Request, _ *interface{}, reply *string) error {
	*reply = NetworkID
	return nil
}
