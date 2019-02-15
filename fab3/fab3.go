/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package fab3

import (
	"context"
	"fmt"
	"net/http"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/gorilla/rpc/v2"
)

type Fab3 struct {
	RPCServer  *rpc.Server
	HTTPServer *http.Server
}

func NewFab3(service EthService, port int) *Fab3 {
	rpcServer := rpc.NewServer()

	proxy := &Fab3{
		RPCServer: rpcServer,
	}

	rpcServer.RegisterCodec(NewRPCCodec(), "application/json")
	msg := "this panic indicates a programming error, and is unreachable"
	if err := rpcServer.RegisterService(service, "eth"); err != nil {
		panic(msg)
	}
	if err := rpcServer.RegisterService(&NetService{}, "net"); err != nil {
		panic(msg)
	}

	r := mux.NewRouter()
	r.Handle("/", proxy.RPCServer)

	allowedHeaders := handlers.AllowedHeaders([]string{"Origin", "Content-Type"})
	allowedOrigins := handlers.AllowedOrigins([]string{"*"})
	allowedMethods := handlers.AllowedMethods([]string{"POST"})

	proxy.HTTPServer = &http.Server{Handler: handlers.CORS(allowedHeaders, allowedOrigins, allowedMethods)(r), Addr: fmt.Sprintf(":%d", port)}
	return proxy
}

func (p *Fab3) Start() error {
	return p.HTTPServer.ListenAndServe()
}

func (p *Fab3) Shutdown() error {
	if p.HTTPServer != nil {
		return p.HTTPServer.Shutdown(context.Background())
	}
	return nil
}
