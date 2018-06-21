# Hyperledger Fabric EVM chaincode plugin

This is the project for the Hyperledger Fabric chaincode plugin for the
Burrow EVM. At its essence, this project enables one to use the Hyperledger
Fabric permissioned blockchain platform to interact with Ethereum smart
contracts written in an EVM compatible language such as Solidity or Viper.

The first phase of this project delivers the chaincode plugin. This chaincode
plugin integrates the Hyperledger Burrow EVM package in a Go chaincode shim
and maps the various methods between the peer and the EVM itself.

The second phase, which should deliver around the time of the Hyperledger fabric
1.3 release in September, 2018 will add a Fabric proxy that implements an
Ethereum compliant JSON RPC interfaces, so that users could use tools, such as
Remix, Truffle, etc, to interact with smart contract running in the Fabric EVM.

We hang out on the
[#fabric-evm channel](https://chat.hyperledger.org/channel/fabric-evm). We are
always interested in feedback and for help in development and testing! See the
[Fabric's CONTRIBUTING documentation](http://hyperledger-fabric.readthedocs.io/en/latest/CONTRIBUTING.html)
for information on how to contribute to this repository.

## Design Document

Please see the design document in [FAB-6590](https://jira.hyperledger.org/browse/FAB-6590).

## Building the hyperledger/fabric-peer with installed plugin image

The easiest way to consume the Fabric EVM chaincode plugin is to either pull the `hyperledger/fabric-peer-evm:amd64-latest` docker image from DockerHub. This is
a `hyperledger/fabric-peer` image with the EVM chaincode plugin pre-installed:

```
docker pull hyperledger/fabric-peer-evm:amd64-latest
```

Or, you can build the same image from this repository if you need a different
architecture. We will be publishing additional architectures as our CI matures.

```
mkdir -p $GOPATH/src/github.com/hyperledger
cd $GOPATH/src/github.com/hyperledger
git clone https://github.com/hyperledger/fabric-chaincode-evm.git
cd fabric-chaincode-evm
git checkout master
make docker
```

You can run the end-2-end test to ensure you have a stable image.

```
make integration-test
```

Once you have an image, you can use that wherever you would normally use the `hyperledger/fabric-peer` image. The end-2-end test above is derivative of the hyperledger/fabric/examples/e2e_cli test. You can compare them to see what is
different.

To interact with the EVM (deploying a contract, executing transactions against
that contract), you will need to leverage one of the Fabric SDKs for the time
being while we continue development of the Web3-Fabric proxy. You can see from
the [e2e_cli/scripts/script.sh](https://github.com/hyperledger/fabric-chaincode-evm/blob/master/e2e_cli/scripts/script.sh)
script in the end-2-end test how to interact with the EVM chaincode using the
Fabric CLI.

Basically, the interaction is the same as with any other chaincode, except that
the first argument of a chaincode invoke is the address for the contract.

[![Creative Commons License](https://i.creativecommons.org/l/by/4.0/88x31.png)](http://creativecommons.org/licenses/by/4.0/)<br>
This work is licensed under a [Creative Commons Attribution 4.0 International License](http://creativecommons.org/licenses/by/4.0/)
