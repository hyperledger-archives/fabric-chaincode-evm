/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package fabproxy

import (
	"context"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/gorilla/rpc/v2"
)

type FabProxy struct {
	rpcServer  *rpc.Server
	httpServer *http.Server
}

func NewFabProxy(service EthService) *FabProxy {
	rpcServer := rpc.NewServer()

	proxy := &FabProxy{
		rpcServer: rpcServer,
	}

	rpcServer.RegisterCodec(NewRPCCodec(), "application/json")
	rpcServer.RegisterService(service, "eth")

	return proxy
}

func (p *FabProxy) Start(port int) {
	r := mux.NewRouter()
	r.Handle("/", p.rpcServer)

	p.httpServer = &http.Server{Handler: r, Addr: fmt.Sprintf(":%d", port)}
	p.httpServer.ListenAndServe()
}

func (p *FabProxy) Shutdown() error {
	return p.httpServer.Shutdown(context.Background())
}
