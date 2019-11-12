module github.com/hyperledger/fabric-chaincode-evm/integration

require (
	code.cloudfoundry.org/clock v0.0.0-20180518195852-02e53af36e6c // indirect
	github.com/containerd/continuity v0.0.0-20181003075958-be9bd761db19 // indirect
	github.com/fsouza/go-dockerclient v1.3.0
	github.com/hyperledger/fabric v1.4.0
	github.com/hyperledger/fabric-chaincode-evm/fab3 v0.0.0
	github.com/onsi/ginkgo v1.10.2
	github.com/onsi/gomega v1.7.0
	github.com/tedsuo/ifrit v0.0.0-20191009134036-9a97d0632f00
	github.com/willf/bitset v1.1.10 // indirect
	go.etcd.io/etcd v3.3.17+incompatible // indirect
)

replace github.com/hyperledger/fabric-chaincode-evm/fab3 => ../fab3

replace github.com/hyperledger/fabric-chaincode-evm/evmcc => ../evmcc

replace github.com/go-kit/kit => github.com/go-kit/kit v0.7.0

replace github.com/hyperledger/burrow => github.com/hyperledger/burrow v0.24.4

replace github.com/fsouza/go-dockerclient => github.com/fsouza/go-dockerclient v1.3.0
