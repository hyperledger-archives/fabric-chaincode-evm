# Fab3 Instruction Set

Fab3 is a partial implementation of the [Ethereum JSON RPC API](https://github.com/ethereum/wiki/wiki/JSON-RPC).
Requests are expected in the following format and must always have a POST header.

```
curl http://127.0.0.1:5000 -X POST -H "Content-Type:application/json" -d '{
  "jsonrpc":"2.0",
  "method": <method>,
  "id":<client-generated-id>,
  "params":<method-params>
}'
```
The examples below use a SimpleStorage contract.
Fab3 currently supports:
- [net_version](#net_version)
- [eth_getCode](#eth_getCode)
- [eth_call](#eth_call)
- [eth_sendTransaction](#eth_sendTransaction)
- [eth_accounts](#eth_accounts)
- [eth_estimateGas](#eth_estimateGas)
- [eth_getBalance](#eth_getBalance)
- [eth_getBlockByNumber](#eth_getBlockByNumber)
- [eth_blockNumber](#eth_blockNumber)
- [eth_getTransactionByHash](#eth_getTransactionByHash)
- [eth_getTransactionReceipt](#eth_getTransactionReceipt)
- [eth_getLogs](#eth_getLogs)
- [eth_getTransactionCount](#eth_getTransactionCount)

### net_version
`net_version` always returns the string `66616265766d`, which is the hex encoding
of `fabevm`. According to the spec, [net_version](https://github.com/ethereum/wiki/wiki/JSON-RPC#net_version)
does not take any parameters.

**Example**
```
curl http://127.0.0.1:5000 -X POST -H "Content-Type:application/json" -d '{
  "jsonrpc":"2.0",
  "method": "net_version",
  "id":1,
  "params":[]
}'

{"jsonrpc":"2.0","result":"66616265766d","id":1}
```


### eth_getCode
`eth_getCode` returns the runtime bytecode of the provided contract address.
According to the spec, [getCode](https://github.com/ethereum/wiki/wiki/JSON-RPC#eth_getcode)
takes in two arguments, the first is the contract address and the second is the
block number specifying the state of the ledger to run the query.

Fab3 does not support querying the state at a certain point in the ledger so the
second argument, if provided, will be ignored. Only the first argument, the
contract address, is required and honored by fab3.

**Example**
```
curl http://127.0.0.1:5000 -X POST -H "Content-Type:application/json" -d '{
  "jsonrpc":"2.0",
  "method": "eth_getCode",
  "id":1,
  "params":["0x40421fd8b64e91da48e703ea1daa488b44ff9d16"]
}'

{"jsonrpc":"2.0","result":"6080604052600436106043576000357c01000000000000000000000000000000000000000000000000000000009004806360fe47b11460485780636d4ce63c14607f575b600080fd5b348015605357600080fd5b50607d60048036036020811015606857600080fd5b810190808035906020019092919050505060a7565b005b348015608a57600080fd5b50609160b1565b6040518082815260200191505060405180910390f35b8060008190555050565b6000805490509056fea165627a7a72305820290b24d16ffaf96310c5e236cef6f8bd81744b72beaeae1ca817d9372b69c2ba0029","id":1}
```

### eth_call
`eth_call` queries the deployed EVMCC and simulates the transaction associated
with the specified parameters. According to the spec, [call](https://github.com/ethereum/wiki/wiki/JSON-RPC#eth_call)
takes in two arguments, the first is an object specifying the parameters of the
transaction and the second is the block number specifying the state of the
ledger to run the query against.

Only the first object is required and honored by fab3. The fields `to`, `data`
are the only fields that are required in the object and the rest are ignored if
provided.

**Example**
```
curl http://127.0.0.1:5000 -X POST -H "Content-Type:application/json" -d '{
  "jsonrpc":"2.0",
  "method": "eth_call",
  "id":1,
  "params":[{"to":"0x40421fd8b64e91da48e703ea1daa488b44ff9d16", "data":"0x6d4ce63c"}]
}'

{"jsonrpc":"2.0","result":"0x000000000000000000000000000000000000000000000000000000000000000a","id":1}
```

### eth_sendTransaction
`eth_sendTransaction` submits a transaction to the EVMCC with the specified
parameters. According to the spec, [sendTransaction](https://github.com/ethereum/wiki/wiki/JSON-RPC#eth_sendtransaction)
takes an object specifying the parameters of the transaction. The fields `to`,
`data` are the only fields that are required in the object and the rest are
ignored if provided. The Fabric transaction id associated to the transaction is
returned.

**Example**
```
curl http://127.0.0.1:5000 -X POST -H "Content-Type:application/json" -d '{
  "jsonrpc":"2.0",
  "method": "eth_sendTransaction",
  "id":1,
  "params":[
    {"to":"0x40421fd8b64e91da48e703ea1daa488b44ff9d16",
    "data":"0x60fe47b1000000000000000000000000000000000000000000000000000000000000000f"}]
}'

{"jsonrpc":"2.0","result":"9807a7ff4ed1962e9414b04f9dec7e05112382a6d826b7e64628fb7f12632dc5","id":1}
```

### eth_accounts
`eth_accounts` queries the EVMCC for the address that is generated from the user
associated to the fab3 instance. The return value will always only have one
address. According to the spec, [accounts](https://github.com/ethereum/wiki/wiki/JSON-RPC#eth_accounts)
does not take any parameters. Note that the returned address is generated on the fly
by the EVMCC and is not stored in the ledger. This should not affect Ethereum
smart contract execution. If a contract stores a user account, it will be
stored under that contract's data.

**Example**
```
curl http://127.0.0.1:5000 -X POST -H "Content-Type:application/json" -d '{
  "jsonrpc":"2.0",
  "method": "eth_accounts",
  "id":1,
  "params":[]
}'

{"jsonrpc":"2.0","result":["0x564fbd2e6e26ca8dbbac758f9253dd80d90974b6"],"id":1}
```

### eth_estimateGas
Gas is hardcoded in the EVMCC and enough is provided for transactions to
complete. Therefore `eth_estimateGas` will always return 0. According to the
spec, [estimateGas](https://github.com/ethereum/wiki/wiki/JSON-RPC#eth_estimategas)
takes in parameters similar to `eth_call`. These parameters, if provided, will
be ignored. However, if the parameter is provided it is expected in the object
format, otherwise an error will be returned.

**Example**
```
curl http://127.0.0.1:5000 -X POST -H "Content-Type:application/json" -d '{
  "jsonrpc":"2.0",
  "method": "eth_estimateGas",
  "id":1,
  "params":[
    {"to":"0x40421fd8b64e91da48e703ea1daa488b44ff9d16",
    "data":"0x60fe47b1000000000000000000000000000000000000000000000000000000000000000f"}]
}'

{"jsonrpc":"2.0","result":"0x0","id":1}
```
### eth_getBalance
No Ether or native tokens are created as part of the EVMCC. User accounts do not
have any balances. Therefore `eth_getBalance` will always return 0 regardless of
the parameters that are provided. According to the spec, [getBalance](https://github.com/ethereum/wiki/wiki/JSON-RPC#eth_getbalance)
expects an array of strings, an account address and a block number. These
parameters, if provided, are ignored. However, if parameters are provided they
must be an array of strings, otherwise an error will be returned.

**Example**
```
curl http://127.0.0.1:5000 -X POST -H "Content-Type:application/json" -d '{
  "jsonrpc":"2.0",
  "method": "eth_getBalance",
  "id":1,
  "params":["0x564fbd2e6e26ca8dbbac758f9253dd80d90974b6"]
}'

{"jsonrpc":"2.0","result":"0x0","id":1}
```

### eth_getBlockByNumber
`eth_getBlockByNumber` returns information about the requested block. According
to the spec, [getBlockByNumber](https://github.com/ethereum/wiki/wiki/JSON-RPC#eth_getblockbynumber)
accepts a number that is hex encoded or a default block parameter such as
`latest`, and `earliest` and a second parameter which is a boolean that
indicates whether full transaction information should be returned. Fabric does
not have a concept of `pending` blocks so providing `pending` as the block
number will result in an error. The field `gasLimit` is provided as a
compatibility measure, and is always hardcoded to `0x0`.

**Example**
```
curl http://127.0.0.1:5000 -X POST -H "Content-Type:application/json" -d '{
  "jsonrpc":"2.0",
  "method": "eth_getBlockByNumber",
  "id":1,
  "params":["latest", true]
}'

{
  "jsonrpc": "2.0",
  "result": {
    "number": "0x8",
    "hash": "0xe63104fc910f90f4d281dbc9d666225d74c5a4ac1438890b4252236d52e158e0",
    "parentHash": "0x8230fad38e199e014aa7433656f78a9d8336ddd6aace9791a6cbbb78c6b9640e",
    "gasLimit": "0x0",
    "transactions": [
      {
        "blockHash": "0xe63104fc910f90f4d281dbc9d666225d74c5a4ac1438890b4252236d52e158e0",
        "blockNumber": "0x8",
        "to": "0x96036d93a9fd3f4cc4cc92e3b9fdb4213f552a99",
        "from": "0xa6e427512d418a9f8f1277dff45a1942236005d3",
        "input": "0x60fe47b1000000000000000000000000000000000000000000000000000000000000000a",
        "transactionIndex": "0x0",
        "hash": "0x1eafc293bd6c4c19dbd965dfb442a1817d2f7b1eaa8fd575a4409539086978dc",
        "gasPrice": "0x0",
        "value": "0x0"
      }
    ]
  },
  "id": 1
}
```

### eth_blockNumber
`eth_blockNumber` returns the number associated with the latest block on the
ledger. According to the spec, [blockNumber](https://github.com/ethereum/wiki/wiki/JSON-RPC#eth_blockNumber)
does not take any parameters.

**Example**
```
curl http://127.0.0.1:5000 -X POST -H "Content-Type:application/json" -d '{
  "jsonrpc":"2.0",
  "method": "eth_BlockNumber",
  "id":1,
  "params":[]
}'

{"jsonrpc":"2.0","result":"0x6","id":1}
```

### eth_getTransactionByHash
`eth_getTransactionByHash` will return transaction information about the given
Fabric transaction id. According to the spec, [getTransactionByHash](https://github.com/ethereum/wiki/wiki/JSON-RPC#eth_gettransactionbyhash)
accepts only one argument, the transaction id.

**Example**
```
curl http://127.0.0.1:5000 -X POST -H "Content-Type:application/json" -d '{
  "jsonrpc":"2.0",
  "method": "eth_getTransactionByHash",
  "id":1,
  "params":["0x9c109bd5880c85a053bee913e82b2395510ef91be133dac06ebf1240e3d73abc"]
}'

{
  "jsonrpc": "2.0",
  "result": {
    "blockHash": "0xffe091745796ce5f1b5cee98fca7ee8a53b5b7c834cdebe74c808a2a5cfbb510",
    "blockNumber": "0x4",
    "to": "0x0000000000000000000000000000000000000000",
    "from": "0xa6e427512d418a9f8f1277dff45a1942236005d3",
    "input": "0x608060405234801561001057600080fd5b5060e68061001f6000396000f3fe6080604052600436106043576000357c01000000000000000000000000000000000000000000000000000000009004806360fe47b11460485780636d4ce63c14607f575b600080fd5b348015605357600080fd5b50607d60048036036020811015606857600080fd5b810190808035906020019092919050505060a7565b005b348015608a57600080fd5b50609160b1565b6040518082815260200191505060405180910390f35b8060008190555050565b6000805490509056fea165627a7a72305820290b24d16ffaf96310c5e236cef6f8bd81744b72beaeae1ca817d9372b69c2ba0029",
    "transactionIndex": "0x0",
    "hash": "0x9c109bd5880c85a053bee913e82b2395510ef91be133dac06ebf1240e3d73abc",
    "gasPrice": "0x0",
    "value": "0x0"
  },
  "id": 1
}
```

### eth_getTransactionReceipt
`eth_getTransactionReceipt` returns the receipt for the transaction. This
includes any logs that were generated from the transaction. If the transaction
was a contract creation, it will return the contract address of the newly
created contract. Otherwise the contract address will be null. According to the
spec, [getTransactionReceipt](https://github.com/ethereum/wiki/wiki/JSON-RPC#eth_gettransactionreceipt)
accepts only one parameter the Fabric transaction id.

**Example**
```
curl http://127.0.0.1:5000 -X POST -H "Content-Type:application/json" -d '{
  "jsonrpc":"2.0",
  "method": "eth_getTransactionReceipt",
  "id":1,
  "params":["0x9c109bd5880c85a053bee913e82b2395510ef91be133dac06ebf1240e3d73abc"]
}'

{
  "jsonrpc": "2.0",
  "result": {
    "transactionHash": "0x9c109bd5880c85a053bee913e82b2395510ef91be133dac06ebf1240e3d73abc",
    "transactionIndex": "0x0",
    "blockHash": "0xffe091745796ce5f1b5cee98fca7ee8a53b5b7c834cdebe74c808a2a5cfbb510",
    "blockNumber": "0x4",
    "contractAddress": "0xad72cffcba95abedf4656a65a2ebab448aae8c19",
    "gasUsed": 0,
    "cumulativeGasUsed": 0,
    "to": "",
    "logs": [],
    "status": "0x1",
    "from": "0xa6e427512d418a9f8f1277dff45a1942236005d3"
  },
  "id": 1
}
```

### eth_getLogs
`eth_getLogs` returns matching log objects from transactions within the matching range of
blocks. These log objects are conversions from the fabric event objects on each transaction. All
visible events will be matched against, which will include other instances of the EVM chaincode
operating with different chaincode IDs.  According to the spec,
[getLogs](https://github.com/ethereum/wiki/wiki/JSON-RPC#eth_getlogs) takes 5 arguments. All are
optional and BlockHash cannot be combined with FromBlock or ToBlock. FromBlock and ToBlock are used
to specify the inclusive range of blocks to search for matching log objects. FromBlock and ToBlock
accept a number that is hex encoded or a default block parameter such as `latest`, and
`earliest`. Fabric does not have a concept of `pending` blocks so providing `pending` as the block
number will result in an error. Address is an individual address or array of addresses which must
match the entries in the log objects. Topics is an array of matching topics which must match the
entries of the log objects.  See the [spec for the
format](https://github.com/ethereum/wiki/wiki/JSON-RPC#a-note-on-specifying-topic-filters) of topic
filters. BlockHash is the exact hash of a fabric block, which will be the only block searched for
transactions that contain matching log entries.

**Example**
```
curl http://127.0.0.1:5000 -X POST -H "Content-Type:application/json" -d '{
  "jsonrpc":"2.0",
  "method": "eth_getLogs",
  "id":1,
  "params":[
    {"fromBlock":"earliest",
    "address":"0x96036d93a9fd3f4cc4cc92e3b9fdb4213f552a99",
    "topics":["0x208509800c5cb9707f116ef96a1d456499ab9fa3c8edc1cdf381fe5216d5b173"]}]
}'

{
  "jsonrpc": "2.0",
  "result": [
    {
      "address": "0x96036d93a9fd3f4cc4cc92e3b9fdb4213f552a99",
      "topics": [
        "0x208509800c5cb9707f116ef96a1d456499ab9fa3c8edc1cdf381fe5216d5b173"
      ],
      "data": "0x000000000000000000000000000000000000000000000000000000000000000a",
      "blockNumber": "0x8",
      "transactionHash": "0x1eafc293bd6c4c19dbd965dfb442a1817d2f7b1eaa8fd575a4409539086978dc",
      "transactionIndex": "0x0",
      "blockHash": "0xe63104fc910f90f4d281dbc9d666225d74c5a4ac1438890b4252236d52e158e0",
      "logIndex": "0x0"
    }
  ],
  "id": 1
}
```

### eth_getTransactionCount
Transaction count per user is not tracked and there is no concept of a nonce in
Fabric. Therefore `eth_getTransactionCount` is hardcoded to always return `0x0`.
According to the spec, [getTransactionCount](https://github.com/ethereum/wiki/wiki/JSON-RPC#eth_gettransactioncount)
takes in an address and a block number. These parameters, if provided will be
ignored.

**Example**
```
curl http://127.0.0.1:5000 -X POST -H "Content-Type:application/json" -d '{
  "jsonrpc":"2.0",
  "method": "eth_getTransactionCount",
  "id":1,
  "params":[]
}'

{"jsonrpc":"2.0","result":"0x0","id":1}
```
