/*
Copyright IBM Corp. All Rights Reserved.
SPDX-License-Identifier: Apache-2.0
*/

vote = require('./voting_contract.js')
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
  var contractAddress = user1.eth.getTransactionReceipt(deployedContract.transactionHash).contractAddress
  var user1Contract = user1VotingContract.at(contractAddress)
  var user2Contract = user2VotingContract.at(contractAddress)

  deployedRuntimeCode = user1.eth.getCode(contractAddress)
  if (deployedRuntimeCode != vote.runtimeVotingContract){
    console.log("Failed to deploy smart contract")
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
  CheckProposal(user1Contract.proposals('0'), 'a',2)
  CheckProposal(user1Contract.proposals('1'), 'b',0)

  console.log("Successfully able to deploy Voting Smart Contract and interact with it")
  process.exit(0)
}

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

console.log("Starting Web3 E2E Test")
// node web3_e2e_test.js addr1 addr2
var user1Address = process.argv[2]
var user2Address = process.argv[3]

TestVotingContract(user1Address, user2Address)
