// Copyright 2017 Monax Industries Limited
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package keys

import (
	"context"
	"fmt"
	"time"

	"github.com/hyperledger/burrow/crypto"
	"github.com/hyperledger/burrow/logging"
	"google.golang.org/grpc"
)

type KeyClient interface {
	// Sign returns the signature bytes for given message signed with the key associated with signAddress
	Sign(signAddress crypto.Address, message []byte) (*crypto.Signature, error)

	// PublicKey returns the public key associated with a given address
	PublicKey(address crypto.Address) (publicKey crypto.PublicKey, err error)

	// Generate requests that a key be generate within the keys instance and returns the address
	Generate(keyName string, keyType crypto.CurveType) (keyAddress crypto.Address, err error)

	// Get the address for a keyname or the adress itself
	GetAddressForKeyName(keyName string) (keyAddress crypto.Address, err error)

	// Returns nil if the keys instance is healthy, error otherwise
	HealthCheck() error
}

var _ KeyClient = (*localKeyClient)(nil)
var _ KeyClient = (*remoteKeyClient)(nil)

type localKeyClient struct {
	ks     *KeyStore
	logger *logging.Logger
}

type remoteKeyClient struct {
	rpcAddress string
	kc         KeysClient
	logger     *logging.Logger
}

func (l *localKeyClient) Sign(signAddress crypto.Address, message []byte) (*crypto.Signature, error) {
	resp, err := l.ks.Sign(nil, &SignRequest{Address: signAddress.String(), Message: message})
	if err != nil {
		return nil, err
	}
	return resp.GetSignature(), nil
}

func (l *localKeyClient) PublicKey(address crypto.Address) (publicKey crypto.PublicKey, err error) {
	resp, err := l.ks.PublicKey(nil, &PubRequest{Address: address.String()})
	if err != nil {
		return crypto.PublicKey{}, err
	}
	curveType, err := crypto.CurveTypeFromString(resp.GetCurveType())
	if err != nil {
		return crypto.PublicKey{}, err
	}
	return crypto.PublicKeyFromBytes(resp.GetPublicKey(), curveType)
}

// Generate requests that a key be generate within the keys instance and returns the address
func (l *localKeyClient) Generate(keyName string, curveType crypto.CurveType) (keyAddress crypto.Address, err error) {
	resp, err := l.ks.GenerateKey(nil, &GenRequest{KeyName: keyName, CurveType: curveType.String()})
	if err != nil {
		return crypto.Address{}, err
	}
	return crypto.AddressFromHexString(resp.GetAddress())
}

func (l *localKeyClient) GetAddressForKeyName(keyName string) (keyAddress crypto.Address, err error) {
	keyAddress, err = crypto.AddressFromHexString(keyName)
	if err == nil {
		return
	}

	all, err := l.ks.GetAllNames()

	if err != nil {
		return crypto.Address{}, err
	}

	if addr, ok := all[keyName]; ok {
		return crypto.AddressFromHexString(addr)
	}

	return crypto.Address{}, fmt.Errorf("`%s` is neither an address or a known key name", keyName)
}

// Returns nil if the keys instance is healthy, error otherwise
func (l *localKeyClient) HealthCheck() error {
	return nil
}

func (l *remoteKeyClient) Sign(signAddress crypto.Address, message []byte) (*crypto.Signature, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	req := SignRequest{Address: signAddress.String(), Message: message}
	l.logger.TraceMsg("Sending Sign request to remote key server: ", fmt.Sprintf("%v", req))
	resp, err := l.kc.Sign(ctx, &req)
	if err != nil {
		l.logger.TraceMsg("Received Sign request error response: ", err)
		return nil, err
	}
	return resp.GetSignature(), nil
}

func (l *remoteKeyClient) PublicKey(address crypto.Address) (publicKey crypto.PublicKey, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	req := PubRequest{Address: address.String()}
	l.logger.TraceMsg("Sending PublicKey request to remote key server: ", fmt.Sprintf("%v", req))
	resp, err := l.kc.PublicKey(ctx, &req)
	if err != nil {
		l.logger.TraceMsg("Received PublicKey error response: ", err)
		return crypto.PublicKey{}, err
	}
	curveType, err := crypto.CurveTypeFromString(resp.GetCurveType())
	if err != nil {
		return crypto.PublicKey{}, err
	}
	l.logger.TraceMsg("Received PublicKey response to remote key server: ", fmt.Sprintf("%v", resp))
	return crypto.PublicKeyFromBytes(resp.GetPublicKey(), curveType)
}

// Generate requests that a key be generate within the keys instance and returns the address
func (l *remoteKeyClient) Generate(keyName string, curveType crypto.CurveType) (keyAddress crypto.Address, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	req := GenRequest{KeyName: keyName, CurveType: curveType.String()}
	l.logger.TraceMsg("Sending Generate request to remote key server: ", fmt.Sprintf("%v", req))
	resp, err := l.kc.GenerateKey(ctx, &req)
	if err != nil {
		l.logger.TraceMsg("Received Generate error response: ", err)
		return crypto.Address{}, err
	}
	l.logger.TraceMsg("Received Generate response to remote key server: ", fmt.Sprintf("%v", resp))
	return crypto.AddressFromHexString(resp.GetAddress())
}

func (l *remoteKeyClient) GetAddressForKeyName(keyName string) (keyAddress crypto.Address, err error) {
	keyAddress, err = crypto.AddressFromHexString(keyName)
	if err == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	key, err := l.kc.List(ctx, &ListRequest{KeyName: keyName})
	if err != nil {
		return crypto.Address{}, err
	}

	if len(key.Key) == 1 {
		return crypto.AddressFromHexString(key.Key[0].Address)
	}

	return crypto.Address{}, fmt.Errorf("`%s` is neither an address or a known key name", keyName)
}

// Returns nil if the keys instance is healthy, error otherwise
func (l *remoteKeyClient) HealthCheck() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_, err := l.kc.List(ctx, &ListRequest{})
	return err
}

// keyClient.New returns a new monax-keys client for provided rpc location
// Monax-keys connects over http request-responses
func NewRemoteKeyClient(rpcAddress string, logger *logging.Logger) (KeyClient, error) {
	logger = logger.WithScope("RemoteKeyClient")
	var opts []grpc.DialOption
	opts = append(opts, grpc.WithInsecure())
	conn, err := grpc.Dial(rpcAddress, opts...)
	if err != nil {
		return nil, err
	}
	kc := NewKeysClient(conn)

	return &remoteKeyClient{kc: kc, rpcAddress: rpcAddress, logger: logger}, nil
}

func NewLocalKeyClient(ks *KeyStore, logger *logging.Logger) KeyClient {
	logger = logger.WithScope("LocalKeyClient")
	return &localKeyClient{ks: ks, logger: logger}
}

type Signer struct {
	keyClient KeyClient
	address   crypto.Address
	publicKey crypto.PublicKey
}

// Creates a AddressableSigner that assumes the address holds an Ed25519 key
func AddressableSigner(keyClient KeyClient, address crypto.Address) (*Signer, error) {
	publicKey, err := keyClient.PublicKey(address)
	if err != nil {
		return nil, err
	}
	// TODO: we can do better than this and return a typed signature when we reform the keys service
	return &Signer{
		keyClient: keyClient,
		address:   address,
		publicKey: publicKey,
	}, nil
}

func (ms *Signer) GetAddress() crypto.Address {
	return ms.address
}

func (ms *Signer) GetPublicKey() crypto.PublicKey {
	return ms.publicKey
}

func (ms *Signer) Sign(messsage []byte) (*crypto.Signature, error) {
	return ms.keyClient.Sign(ms.address, messsage)
}
