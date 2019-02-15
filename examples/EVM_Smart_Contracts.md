# EVM Smart Contracts
The Ethereum Virtual Machine (EVM) is a spec of a limited instruction set that
has been used to run smart contracts in the Ethereum networks. The EVM that
was created through the [Hyperledger Burrow](https://github.com/hyperledger/burrow)
project and has been integrated into Fabric, allowing deployment on contracts
that can be compiled into EVM bytecode.

The EVM is installed into Fabric as a user chaincode and then smart contracts can
be deployed through that. A single EVM chaincode is enough to run multiple ethereum
smart contracts on a channel. The chaincode does not adopt ethereum's method of
consensus. All transactions will still follow the execute, order, validate steps
in the Fabric transaction flow. Be sure to install the chaincode on enough peers in
different orgs and set an endorsement policy that ensures a degree of
decentralization. In order to interact with the smart contracts that have been
deployed there is a `fab3` which implements a limited set of APIs from the
Ethereum JSON RPC API and therefore can be used as a web3 provider.

## Installing the EVM Chaincode
The EVM chaincode is located in the [fabric-chaincode-evm](https://github.com/hyperledger/fabric-chaincode-evm)
repo under `evmcc`. To install the chaincode follow the usual steps to install a chaincode. The following
instructions are based on the version 1.3 of `first-network` tutorial in the [fabric-samples](https://github.com/hyperledger/fabric-samples).


### Mount the EVM Chaincode
Update the ``docker-compose-cli.yaml`` with the volumes to include the ``fabric-chaincode-evm``.

```yaml
  cli:
    volumes:
      - ./../../fabric-chaincode-evm:/opt/gopath/src/github.com/hyperledger/fabric-chaincode-evm
```

Start the network by running:

```bash
  ./byfn up
```

### Build and Start the EVM


```bash
  docker exec -it cli bash
```

If successful, you should see the following prompt

```bash
  root@0d78bb69300d:/opt/gopath/src/github.com/hyperledger/fabric/peer#
```

To change which peer is targeted change the following environment variables:

```bash
  # Environment variables for PEER0
  export CORE_PEER_MSPCONFIGPATH=/opt/gopath/src/github.com/hyperledger/fabric/peer/crypto/peerOrganizations/org1.example.com/users/Admin@org1.example.com/msp
  export CORE_PEER_ADDRESS=peer0.org1.example.com:7051
  export CORE_PEER_LOCALMSPID="Org1MSP"
  export CORE_PEER_TLS_ROOTCERT_FILE=/opt/gopath/src/github.com/hyperledger/fabric/peer/crypto/peerOrganizations/org1.example.com/peers/peer0.org1.example.com/tls/ca.crt
```

Next install the EVM chaincode on all the peers
```bash
    peer chaincode install -n evmcc -l golang -v 0 -p github.com/hyperledger/fabric-chaincode-evm/evmcc
```

Instantiate the evmcc and replace ``<channel-name>`` with the channel name

```bash
    peer chaincode instantiate -n evmcc -v 0 -C <channel-name> -c '{"Args":[]}' -o orderer.example.com:7050 --tls --cafile /opt/gopath/src/github.com/hyperledger/fabric/peer/crypto/ordererOrganizations/example.com/orderers/orderer.example.com/msp/tlscacerts/tlsca.example.com-cert.pem
```
## Interact with the EVM Chaincode

There are two general ways to interact with the EVM Chaincode: the usual Fabric tools & Web3

### Using the Peer CLI
In general the evm expects two arguments, the `to` address and the `input` that is necessary in ethereum transactions.

The following is an example that deploys and interacts with the [Simple Storage](https://solidity.readthedocs.io/en/v0.4.24/introduction-to-smart-contracts.html) contract.

#### Deploying a Contract
To deploy smart contracts the `to` field is the zero address and the `input` is the compiled evm bytecode of the contract.

```bash
  peer chaincode invoke -n evmcc -C <channel-name>  -c '{"Args":["0000000000000000000000000000000000000000","608060405234801561001057600080fd5b5060df8061001f6000396000f3006080604052600436106049576000357c0100000000000000000000000000000000000000000000000000000000900463ffffffff16806360fe47b114604e5780636d4ce63c146078575b600080fd5b348015605957600080fd5b5060766004803603810190808035906020019092919050505060a0565b005b348015608357600080fd5b50608a60aa565b6040518082815260200191505060405180910390f35b8060008190555050565b600080549050905600a165627a7a723058203dbaed52da8059a841ed6d7b484bf6fa6f61a7e975a803fdedf076a121a8c4010029"]}' -o orderer.example.com:7050 --tls --cafile /opt/gopath/src/github.com/hyperledger/fabric/peer/crypto/ordererOrganizations/example.com/orderers/orderer.example.com/msp/tlscacerts/tlsca.example.com-cert.pem
```

The payload of that transaction will be the contract address for your deployed contract.
To verify that your contract has deployed successful you can query the `evmcc` for the runtime bytecode of the contract:

```bash
  peer chaincode query -n evmcc -C <channel-name> -c '{"Args":["getCode","<contract addr>"]}'
```

The payload of that query should return the runtime bytecode which should be the following:

```bash
  6080604052600436106049576000357c0100000000000000000000000000000000000000000000000000000000900463ffffffff16806360fe47b114604e5780636d4ce63c146078575b600080fd5b348015605957600080fd5b5060766004803603810190808035906020019092919050505060a0565b005b348015608357600080fd5b50608a60aa565b6040518082815260200191505060405180910390f35b8060008190555050565b600080549050905600a165627a7a723058203dbaed52da8059a841ed6d7b484bf6fa6f61a7e975a803fdedf076a121a8c4010029
```

#### Interacting with a Deployed Contract
To interact with the deployed smart contract you need to use the contract address that you received in the previous section.

The Simple Storage Contract has two functions, `set(x)` and `get()`. In these transactions the `to` field is the contract address and
the `input` field is the function hash concatenated with any of the required arguments.

Let's first set the value being stored. The function hash for `set` is `60fe47b1` and we want to set the value to 10 then we need to
concatenate the hash with `000000000000000000000000000000000000000000000000000000000000000a`

```bash
  peer chaincode invoke -n evmcc -C <channel-name> -c '{"Args":["<contract-address>","60fe47b1000000000000000000000000000000000000000000000000000000000000000a"]}' -o orderer.example.com:7050 --tls --cafile /opt/gopath/src/github.com/hyperledger/fabric/peer/crypto/ordererOrganizations/example.com/orderers/orderer.example.com/msp/tlscacerts/tlsca.example.com-cert.pem
```

Now to verify that the function was invoked we can query the value by running `get` which has the function hash `6d4ce63c`.

```bash
  peer chaincode query -n evmcc -C <channel-name> -c '{"Args":["<contract-address>","6d4ce63c"]}' --hex
```

The output of that query should result in a payload of ``a``.

#### Getting the User Account Address
As Fabric does not use user accounts, as part of the EVM CC no user account information is stored. However we do have a mechanism to generate
a user account address from the user's public key. This is used for the ``EVMCC`` transactions when needed. We also provide a mechanism information
users to access that address if they would like.

```bash
  peer chaincode query -n evmcc -C <channel-name> -c '{"Args":["account"]}'
```
The payload will be your user address.

### Using Web3
Web3.js is a library that improves the user experience in deploying and managing EVM smart contracts. It expects a provider that has
implemented the [Ethereum JSON RPC API](https://github.com/ethereum/wiki/wiki/JSON-RPC). The Fab Proxy has support for a limited set
the APIs that do allow for using web3. The following should not be done in the cli docker container. It should be done outside where
you would like to run the proxy.

#### Setting up the Fab Proxy
The fabric proxy uses the Fabric Go SDK to connect and interact with the fabric network. To start you will need a SDK config.
This [config](first-network-sdk-config.yaml) will work for the first-network example. The config assumes that the `fabric-samples` repo
is in your `$GOPATH` and that all your certs will be in default location of the `first-network` example.

The proxy depends on a set of Environment variables to work.
```bash
  # Environment Variables for Fab3:
  export FAB3_CONFIG=${GOPATH}/src/github.com/hyperledger/fabric-chaincode-evm/examples/first-network-sdk-config.yaml # Path to a compatible Fabric SDK Go config file
  export FAB3_USER=User1 # User identity being used for the proxy (Matches the users names in the crypto-config directory specified in the config)
  export FAB3_ORG=Org1  # Organization of the specified user
  export FAB3_CHANNEL=mychannel # Channel to be used for the transactions
  export FAB3_CCID=evmcc # ID of the EVM Chaincode deployed in your fabric network. If not provided default is evmcc.
  export FAB3_PORT=5000 # Port the proxy will listen on. If not provided default is 5000.
```
Set the required variables before running the proxy.

#### Building the Fab Proxy
The proxy can be built like other go projects. Make sure you are at the root of this repo and the repo is in your gopath.

```bash
  make fab3
```
You should see a binary `fab3` in the `bin` subdirectory. If you have set the required environment variables you can run the proxy by

```bash
  bin/fab3
```
If you used the default port you should see output like the following:
```
{"level":"info","ts":1550530404.3546276,"logger":"fab3","caller":"cmd/main.go:143","msg":"starting-fab3","port":5000}
```

##### Connecting to the Proxy
The following directions require ``node`` and ``web3`` to be installed. The instructions follow the `web3` api for version `0.20.2`
To install the same version of `web3` run:
```
npm install web3@0.20.2
```

After installing the correct version of `web3`, in a node session run the
following to connect to the proxy:

```
  > Web3 = require('web3')
  ...
  > web3 = new Web3(new Web3.providers.HttpProvider('http://localhost:5000'))
```

If successful you should be able to get your account address. The first query or transaction you run with the proxy will take a little
longer than others since the SDK is using the discovery service to find out about all the peers on the network.

```
  > web3.eth.accounts
```

And you should see an single element array with your account address.
In order to run any transactions web3 requires `web3.eth.defaultAccount` to be set

```
  > web3.eth.defaultAccount = web3.eth.accounts[0]
```

#### Deploying a Smart Contract
This process should be familiar to the Ethereum style of deploying contracts using web3. For the this example we will be using the
[Simple Storage](https://solidity.readthedocs.io/en/v0.4.24/introduction-to-smart-contracts.html) contract.

You will need the compiled evm bytecode and the ABI of the contract to proceed.

```
  > simpleStorageABI = [
  	{
  		"constant": false,
  		"inputs": [
  			{
  				"name": "x",
  				"type": "uint256"
  			}
  		],
  		"name": "set",
  		"outputs": [],
  		"payable": false,
  		"stateMutability": "nonpayable",
  		"type": "function"
  	},
  	{
  		"constant": true,
  		"inputs": [],
  		"name": "get",
  		"outputs": [
  			{
  				"name": "",
  				"type": "uint256"
  			}
  		],
  		"payable": false,
      "stateMutability": "view",
      "type": "function"
  	}
  ]

  > simpleStorageBytecode = '608060405234801561001057600080fd5b5060df8061001f6000396000f3006080604052600436106049576000357c0100000000000000000000000000000000000000000000000000000000900463ffffffff16806360fe47b114604e5780636d4ce63c146078575b600080fd5b348015605957600080fd5b5060766004803603810190808035906020019092919050505060a0565b005b348015608357600080fd5b50608a60aa565b6040518082815260200191505060405180910390f35b8060008190555050565b600080549050905600a165627a7a723058203dbaed52da8059a841ed6d7b484bf6fa6f61a7e975a803fdedf076a121a8c4010029'

  > SimpleStorage = web3.eth.contract(simpleStorageABI)

  > deployedContract = SimpleStorage.new([], {data: simpleStorageBytecode})
  > myContract = SimpleStorage.at(web3.eth.getTransactionReceipt(deployedContract.transactionHash).contractAddress)
```

#### Interacting with a Previously Deployed Contract
If you already had a deployed Simple Storage contract you can create an contract instance using the contract address.
The following assumes you have already created the Simple Storage Object type using the SimpleStorageABI.

```
  > myContract = SimpleStorage.at(<contract-address>)
```
Now lets interact with the contract by setting the value to 10.

```
  > myContract.set(10)
```

To verify that the transaction worked you can query the value set by running `get()`
```
  > myContract.get().toNumber()
```
That should return 10
