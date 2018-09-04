/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package fabproxy

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gorilla/rpc/v2"
	"github.com/gorilla/rpc/v2/json2"
)

type rpcCodec struct {
	codec *json2.Codec
}

type rpcCodecRequest struct {
	rpc.CodecRequest
}

func NewRPCCodec() rpc.Codec {
	return &rpcCodec{codec: json2.NewCodec()}
}

func (c *rpcCodec) NewRequest(r *http.Request) rpc.CodecRequest {
	return &rpcCodecRequest{c.codec.NewRequest(r)}
}

/* Gorilla RPC expects methods follow Go style of service.Method, where service is an object that has the method `Method`.
Ethereum JSON RPC expect methods to follow service_method
String conversion is necessary to change service_method to service.Method to meet Gorilla RPC requirements
*/
func (r *rpcCodecRequest) Method() (string, error) {

	m, err := r.CodecRequest.Method()
	if err != nil {
		return "", err
	}
	method := strings.Split(m, "_")
	if len(method) > 2 {
		return "", fmt.Errorf("Received a malformed method: %s", method)
	}

	modifiedMethod := fmt.Sprintf("%s.%s", method[0], strings.Title(method[1]))
	return modifiedMethod, nil
}
