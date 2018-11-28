/*
Copyright IBM Corp. All Rights Reserved.
SPDX-License-Identifier: Apache-2.0
*/
solidityEvent = require("web3/lib/web3/event.js");
vote = require('./voting_contract.js')
instructor = require('./instructor_contract.js')

Web3 = require('web3')
web3 = new Web3()


function TestVotingContract(fabProxyAddress1, fabProxyAddress2){
  var user1 = new Web3(new Web3.providers.HttpProvider(fabProxyAddress1))
  var user2 = new Web3(new Web3.providers.HttpProvider(fabProxyAddress2))

  // Get user addresses
  var user1Addr = user1.eth.accounts[0]
  if (user1Addr == ""){
    console.log("Unable to query user1's account address")
    process.exit(1)
  }
  user1.eth.defaultAccount = user1Addr

  var user2Addr = user2.eth.accounts[0]
  if (user2Addr == ""){
    console.log("Unable to query user2's account address")
    process.exit(1)
  }
  user2.eth.defaultAccount = user2Addr

  if (user1Addr == user2Addr){
    console.log("Both user accounts are the same")
    process.exit(1)
  }

  // Create and Deploy Contract Objects
  // Each user needs their own version of the contract as the contract object is connected to the account that
  // created the object
  var user1VotingContract = user1.eth.contract(vote.votingContractABI)
  var user2VotingContract = user2.eth.contract(vote.votingContractABI)

  var deployedContract = user1VotingContract.new(['a','b'], {data: vote.compiledVotingContract})
  var contractTransaction = user1.eth.getTransaction(deployedContract.transactionHash)
  if ( ! contractTransaction.input.includes(vote.compiledVotingContract) ){
    console.log("getTransaction should return transaction with full details that includes the compiled contract")
    console.log(contractTransaction)
    console.log(vote.compiledVotingContract)
    console.log(contractTransaction.input)
    process.exit(1)
  }
  var contractTransactionBlock = user1.eth.getBlock(contractTransaction.blockNumber, false)
  if ( contractTransactionBlock.transactions[contractTransaction.transactionIndex] != contractTransaction.hash ) {
    console.log("getBlock should have a block with the same transaction as before")
    console.log(contractTransactionBlock)
    console.log(contractTransaction)
    process.exit(1)
  }

  var contractAddress = user1.eth.getTransactionReceipt(deployedContract.transactionHash).contractAddress
  var user1Contract = user1VotingContract.at(contractAddress)
  var user2Contract = user2VotingContract.at(contractAddress)

  deployedRuntimeCode = user1.eth.getCode(contractAddress)
  if (deployedRuntimeCode != vote.runtimeVotingContract){
    console.log("Failed to deploy Voting smart contract")
    process.exit(1)
  }

  // Check that proposals have been properly initialized
  CheckProposal(user1Contract.proposals('0'), 'a',0)
  CheckProposal(user1Contract.proposals('1'), 'b',0)

  // Interact with the contract
  user1Contract.vote('0')

  // Check that proposals have been properly updated after vote.
  CheckProposal(user1Contract.proposals('0'), 'a',1)
  CheckProposal(user1Contract.proposals('1'), 'b',0)

  // Check that User2 can query the smart contract code
  if (user2.eth.getCode(contractAddress) != vote.runtimeVotingContract){
    console.log("User2 was unable to getCode for the contract")
    process.exit(1)
  }

  // User1 Should give User2 ability to vote
  user1Contract.giveRightToVote(user2.eth.defaultAccount)

  // User2 should be able to vote
  user2Contract.vote('0')

  // Check the votes
  CheckProposal(user1Contract.proposals('0'), 'a',2)
  CheckProposal(user1Contract.proposals('1'), 'b',0)

  console.log("Successfully able to deploy Voting Smart Contract and interact with it")
}

function TestInstructorContractEvents(fabProxyAddress){
  var user = new Web3(new Web3.providers.HttpProvider(fabProxyAddress))

  // Get user addresses
  var userAddr = user.eth.accounts[0]
  if (userAddr == ""){
    console.log("Unable to query user1's account address")
    process.exit(1)
  }
  user.eth.defaultAccount = userAddr

  // Create and Deploy Contract Objects
  var instructorContract = user.eth.contract(instructor.instructorContractABI)

  var deployedContract = instructorContract.new(['a','b'], {data: instructor.compiledInstructorContract})
  var contractAddress = user.eth.getTransactionReceipt(deployedContract.transactionHash).contractAddress
  var userContract = instructorContract.at(contractAddress)

  deployedRuntimeCode = user.eth.getCode(contractAddress)
  if (deployedRuntimeCode != instructor.runtimeInstructorContract){
    console.log("Failed to deploy Instructor smart contract")
    process.exit(1)
  }

  txID = userContract.setInstructor('0x53616d', 25, 30000)
  receipt = user.eth.getTransactionReceipt(txID)
  if (receipt.logs == null) {
    console.log("No logs were found for Instructor Contract transaction")
    process.exit(1)
  }
  var logs = receipt.logs

  if (logs.length != 1) {
    // Instructor Contract should only produce one log object
    console.log("Only one log should exist from instructor contract transaction")
    process.exit(1)
  }

  if (logs[0].topics.length != 2) {
    // Setter Event should two topics, 1. setter signature, 2. value of indexed param
    console.log("Setter event should have two topics")
    process.exit(1)
  }

  if (logs[0].topics[0] != "0x" + instructor.setterEventSignature) {
    console.log("First topic should be the Event Signature: Setter(bytes32,uint,uint)")
    console.log("Expect topic: 0x" + logs[0].topics[0] + " to equal: 0x" + instructor.setterEventSignature )
    process.exit(1)
  }

  EventDecoder(logs, instructor.instructorContractABI)
  decodedEvent = logs[0]

  if (decodedEvent.event != "Setter"){
    console.log("Incorrect Event in decoded event")
    process.exit(1)
  }

  eventArgs = decodedEvent.args
  if (eventArgs.name != '0x53616d0000000000000000000000000000000000000000000000000000000000'){
    console.log("Event has incorrect name")
    console.log("Expected name " + eventArgs.name + " to equal 0x53616d0000000000000000000000000000000000000000000000000000000000")
    process.exit(1)
  }

  if (eventArgs.age.toNumber() != 25){
    console.log("Event has the wrong age")
    console.log("Expected age " + eventArgs.age.toNumber() + " to equal 25")
    process.exit(1)
  }

  if (eventArgs.salary.toNumber() != 30000){
    console.log("Event has the wrong salary")
    console.log("Expected salary " + eventArgs.salary.toNumber() + " to equal 30000")
    process.exit(1)
  }

  console.log("Successfully able to deploy Instructor Smart Contract, see events, and interact with it")
}

//Check Proposal checks the proposal object in the Voting Contract
function CheckProposal(proposal, expectedName, expectedCount){
  var proposalName = web3.toUtf8(proposal[0])
  if ( proposalName != expectedName ){
    console.log("Proposal name: "+ proposalName  + " does not match expected name "+ expectedName)
    process.exit(1)
  }

  if (proposal[1].toNumber() != expectedCount ){
    console.log("Proposal count: "+ proposal[1].toNumber() + " does not match expected count "+ expectedCount)
    process.exit(1)
  }
}

//This function will in place decode the log objects that match events in the abi
function EventDecoder(logs, abi){
  var decoders = abi.filter(function (json) {
    return json.type === 'event';
  }).map(function(json) {
    return new solidityEvent(null, json, null);
  });

  return logs.map(function (log) {
    return decoders.find(function(decoder) {
      return (decoder.signature() == log.topics[0].replace("0x",""));
    }).decode(log);
  })
}


console.log("Starting Web3 E2E Test")
// node web3_e2e_test.js addr1 addr2
var user1Address = process.argv[2]
var user2Address = process.argv[3]

TestVotingContract(user1Address, user2Address)
TestInstructorContractEvents(user1Address)
console.log("Finished Web3 Tests")
