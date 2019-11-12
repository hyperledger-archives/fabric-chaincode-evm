module github.com/hyperledger/fabric-chaincode-evm/fab3

require (
	github.com/cloudflare/cfssl v0.0.0-20180223231731-4e2dcbde5004 // indirect; from fabric-go-sdk
	github.com/go-kit/kit v0.9.0
	github.com/gogo/protobuf v1.3.1
	github.com/golang/mock v1.2.1-0.20190116182947-c20582278a82 // indirect
	github.com/google/certificate-transparency-go v1.0.21 // indirect
	github.com/gorilla/handlers v1.4.2
	github.com/gorilla/mux v1.7.3
	github.com/gorilla/rpc v1.2.0+incompatible
	github.com/hyperledger/burrow v0.24.4
	github.com/hyperledger/fabric v1.4.0
	github.com/hyperledger/fabric-chaincode-evm/evmcc v0.0.0
	github.com/hyperledger/fabric-lib-go v1.0.0 // indirect
	github.com/hyperledger/fabric-sdk-go v1.0.0-alpha5
	github.com/onsi/ginkgo v1.10.2
	github.com/onsi/gomega v1.7.0
	github.com/pkg/errors v0.8.1
	github.com/spf13/cobra v0.0.1
	github.com/spf13/viper v1.4.0
	github.com/zmap/zlint v1.0.2 // indirect
	go.uber.org/zap v1.10.0
)

replace github.com/hyperledger/fabric-chaincode-evm/evmcc => ../evmcc

replace github.com/go-kit/kit => github.com/go-kit/kit v0.8.0
