# Hyperledger Fabric EVM chaincode

This is the project for the Hyperledger Fabric chaincode, integrating the
Burrow EVM. At its essence, this project enables one to use the Hyperledger
Fabric permissioned blockchain platform to interact with Ethereum smart
contracts written in an EVM compatible language such as Solidity or Vyper.

The integration has two main pieces. The chaincode, which integrates the
Hyperledger Burrow EVM package in a Go chaincode shim and maps the various
methods between the peer and the EVM itself.

The second piece is a Fabric Proxy that implements a subset of the Ethereum
compliant JSON RPC interfaces, so that users could use tools such as Web3.js
to interact with smart contracts running in the Fabric EVM. Currently the APIs
that have been implemented are `eth_getCode`, `eth_account`, `eth_call`,
`sendTransaction`,`eth_getTransactionReceipt`. We are working on expanding
that subset.

We hang out in the
[#fabric-evm channel](https://chat.hyperledger.org/channel/fabric-evm). We are
always interested in feedback and help in development and testing! For more
information about contributing look below at the [Contributions](#Contributions)
section.


## Design Document

Please see the design document in [FAB-6590](https://jira.hyperledger.org/browse/FAB-6590).

## Deploying the Fabric EVM Chaincode

This chaincode can be deployed like any other user chaincode to Hyperledger
Fabric. The chaincode has no instantiation arguments.

You can run the integration test in which a sample Fabric Network is run and the
chaincode is installed with the CCID: `evmcc`.
```
make integration-test
```
The end-2-end test is derivative of the hyperledger/fabric/integration/e2e test.
You can compare them to see what is different.

We have an [tutorial](examples/EVM_Smart_Contracts.md) that runs through the
basic setup of the EVM chaincode as well as setting up the Fabric Proxy.

Basically, the interaction is the same as with any other chaincode, except that
the first argument of a chaincode invoke is the address for the contract and
the second argument is the input you typically provide for an Ethereum
transaction.

## Contributions
The `fabric-chaincode-evm` lives in a [gerrit repository](https://gerrit.hyperledger.org/r/#/admin/projects/fabric-chaincode-evm).
The github repository is a mirror. For more information on how to contribute
look at [Fabric's CONTRIBUTING documentation](http://hyperledger-fabric.readthedocs.io/en/latest/CONTRIBUTING.html).

Please send all pull requests to the gerrit repository. For issues, open a ticket in
the Hyperledger Fabric [JIRA](https://jira.hyperledger.org/projects/FAB/issues)
and add `fabric-chaincode-evm` in the component field.

Current Dependencies:
- Hyperledger Fabric [v1.4](https://github.com/hyperledger/fabric/releases/tag/v1.4.0). EVMCC can be run on Fabric 1.0 and newer.
- Hyperledger Fabric SDK Go [revision = "beccd9cb1450fddfe426616e151d709c99f7ccdd"](https://github.com/hyperledger/fabric-sdk-go/tree/beccd9cb1450fddfe426616e151d709c99f7ccdd)
- Dep [v0.5](https://github.com/golang/dep/releases/tag/v0.5.0)
- Minimum of Go 1.10 is required to compile Fab3.

[![Creative Commons License](https://i.creativecommons.org/l/by/4.0/88x31.png)](http://creativecommons.org/licenses/by/4.0/)<br>
This work is licensed under a [Creative Commons Attribution 4.0 International License](http://creativecommons.org/licenses/by/4.0/)
