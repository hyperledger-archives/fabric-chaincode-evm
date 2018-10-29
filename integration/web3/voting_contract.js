/*
Copyright IBM Corp. All Rights Reserved.
SPDX-License-Identifier: Apache-2.0
*/


/* This file has both the compiled contract code, the runtime code and ABI for the Voting Contract

Voting Contract
pragma solidity ^0.4.22;

/// @title Voting with delegation.
contract Ballot {
    // This declares a new complex type which will
    // be used for variables later.
    // It will represent a single voter.
    struct Voter {
        uint weight; // weight is accumulated by delegation
        bool voted;  // if true, that person already voted
        address delegate; // person delegated to
        uint vote;   // index of the voted proposal
    }

    // This is a type for a single proposal.
    struct Proposal {
        bytes32 name;   // short name (up to 32 bytes)
        uint voteCount; // number of accumulated votes
    }

    address public chairperson;

    // This declares a state variable that
    // stores a `Voter` struct for each possible address.
    mapping(address => Voter) public voters;

    // A dynamically-sized array of `Proposal` structs.
    Proposal[] public proposals;

    /// Create a new ballot to choose one of `proposalNames`.
    constructor(bytes32[] proposalNames) public {
        chairperson = msg.sender;
        voters[chairperson].weight = 1;

        // For each of the provided proposal names,
        // create a new proposal object and add it
        // to the end of the array.
        for (uint i = 0; i < proposalNames.length; i++) {
            // `Proposal({...})` creates a temporary
            // Proposal object and `proposals.push(...)`
            // appends it to the end of `proposals`.
            proposals.push(Proposal({
                name: proposalNames[i],
                voteCount: 0
            }));
        }
    }

    // Give `voter` the right to vote on this ballot.
    // May only be called by `chairperson`.
    function giveRightToVote(address voter) public {
        // If the first argument of `require` evaluates
        // to `false`, execution terminates and all
        // changes to the state and to Ether balances
        // are reverted.
        // This used to consume all gas in old EVM versions, but
        // not anymore.
        // It is often a good idea to use `require` to check if
        // functions are called correctly.
        // As a second argument, you can also provide an
        // explanation about what went wrong.
        require(
            msg.sender == chairperson,
            "Only chairperson can give right to vote."
        );
        require(
            !voters[voter].voted,
            "The voter already voted."
        );
        require(voters[voter].weight == 0);
        voters[voter].weight = 1;
    }

    /// Delegate your vote to the voter `to`.
    function delegate(address to) public {
        // assigns reference
        Voter storage sender = voters[msg.sender];
        require(!sender.voted, "You already voted.");

        require(to != msg.sender, "Self-delegation is disallowed.");

        // Forward the delegation as long as
        // `to` also delegated.
        // In general, such loops are very dangerous,
        // because if they run too long, they might
        // need more gas than is available in a block.
        // In this case, the delegation will not be executed,
        // but in other situations, such loops might
        // cause a contract to get "stuck" completely.
        while (voters[to].delegate != address(0)) {
            to = voters[to].delegate;

            // We found a loop in the delegation, not allowed.
            require(to != msg.sender, "Found loop in delegation.");
        }

        // Since `sender` is a reference, this
        // modifies `voters[msg.sender].voted`
        sender.voted = true;
        sender.delegate = to;
        Voter storage delegate_ = voters[to];
        if (delegate_.voted) {
            // If the delegate already voted,
            // directly add to the number of votes
            proposals[delegate_.vote].voteCount += sender.weight;
        } else {
            // If the delegate did not vote yet,
            // add to her weight.
            delegate_.weight += sender.weight;
        }
    }

    /// Give your vote (including votes delegated to you)
    /// to proposal `proposals[proposal].name`.
    function vote(uint proposal) public {
        Voter storage sender = voters[msg.sender];
        require(!sender.voted, "Already voted.");
        sender.voted = true;
        sender.vote = proposal;

        // If `proposal` is out of the range of the array,
        // this will throw automatically and revert all
        // changes.
        proposals[proposal].voteCount += sender.weight;
    }

    /// @dev Computes the winning proposal taking all
    /// previous votes into account.
    function winningProposal() public view
            returns (uint winningProposal_)
    {
        uint winningVoteCount = 0;
        for (uint p = 0; p < proposals.length; p++) {
            if (proposals[p].voteCount > winningVoteCount) {
                winningVoteCount = proposals[p].voteCount;
                winningProposal_ = p;
            }
        }
    }

    // Calls winningProposal() function to get the index
    // of the winner contained in the proposals array and then
    // returns the name of the winner
    function winnerName() public view
            returns (bytes32 winnerName_)
    {
        winnerName_ = proposals[winningProposal()].name;
    }
}
*/

compiledVotingContract = '608060405234801561001057600080fd5b50604051610d8b380380610d8b833981018060405281019080805182019291905050506000336000806101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff16021790555060018060008060009054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002060000181905550600090505b81518110156101735760026040805190810160405280848481518110151561010657fe5b90602001906020020151600019168152602001600081525090806001815401808255809150509060018203906000526020600020906002020160009091929091909150600082015181600001906000191690556020820151816001015550505080806001019150506100e2565b5050610c07806101846000396000f30060806040526004361061008e576000357c0100000000000000000000000000000000000000000000000000000000900463ffffffff1680630121b93f14610093578063013cf08b146100c05780632e4176cf146101105780635c19a95c14610167578063609ff1bd146101aa5780639e7b8d61146101d5578063a3ec138d14610218578063e2ba53f0146102b4575b600080fd5b34801561009f57600080fd5b506100be600480360381019080803590602001909291905050506102e7565b005b3480156100cc57600080fd5b506100eb6004803603810190808035906020019092919050505061040d565b6040518083600019166000191681526020018281526020019250505060405180910390f35b34801561011c57600080fd5b50610125610440565b604051808273ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200191505060405180910390f35b34801561017357600080fd5b506101a8600480360381019080803573ffffffffffffffffffffffffffffffffffffffff169060200190929190505050610465565b005b3480156101b657600080fd5b506101bf61088a565b6040518082815260200191505060405180910390f35b3480156101e157600080fd5b50610216600480360381019080803573ffffffffffffffffffffffffffffffffffffffff169060200190929190505050610905565b005b34801561022457600080fd5b50610259600480360381019080803573ffffffffffffffffffffffffffffffffffffffff169060200190929190505050610b4f565b60405180858152602001841515151581526020018373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200182815260200194505050505060405180910390f35b3480156102c057600080fd5b506102c9610bac565b60405180826000191660001916815260200191505060405180910390f35b6000600160003373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002090508060010160009054906101000a900460ff161515156103b1576040517f08c379a000000000000000000000000000000000000000000000000000000000815260040180806020018281038252600e8152602001807f416c726561647920766f7465642e00000000000000000000000000000000000081525060200191505060405180910390fd5b60018160010160006101000a81548160ff02191690831515021790555081816002018190555080600001546002838154811015156103eb57fe5b9060005260206000209060020201600101600082825401925050819055505050565b60028181548110151561041c57fe5b90600052602060002090600202016000915090508060000154908060010154905082565b6000809054906101000a900473ffffffffffffffffffffffffffffffffffffffff1681565b600080600160003373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002091508160010160009054906101000a900460ff16151515610530576040517f08c379a00000000000000000000000000000000000000000000000000000000081526004018080602001828103825260128152602001807f596f7520616c726561647920766f7465642e000000000000000000000000000081525060200191505060405180910390fd5b3373ffffffffffffffffffffffffffffffffffffffff168373ffffffffffffffffffffffffffffffffffffffff16141515156105d4576040517f08c379a000000000000000000000000000000000000000000000000000000000815260040180806020018281038252601e8152602001807f53656c662d64656c65676174696f6e20697320646973616c6c6f7765642e000081525060200191505060405180910390fd5b5b600073ffffffffffffffffffffffffffffffffffffffff16600160008573ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002060010160019054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1614151561077b57600160008473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002060010160019054906101000a900473ffffffffffffffffffffffffffffffffffffffff1692503373ffffffffffffffffffffffffffffffffffffffff168373ffffffffffffffffffffffffffffffffffffffff1614151515610776576040517f08c379a00000000000000000000000000000000000000000000000000000000081526004018080602001828103825260198152602001807f466f756e64206c6f6f7020696e2064656c65676174696f6e2e0000000000000081525060200191505060405180910390fd5b6105d5565b60018260010160006101000a81548160ff021916908315150217905550828260010160016101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff160217905550600160008473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002090508060010160009054906101000a900460ff161561086e5781600001546002826002015481548110151561084b57fe5b906000526020600020906002020160010160008282540192505081905550610885565b816000015481600001600082825401925050819055505b505050565b6000806000809150600090505b60028054905081101561090057816002828154811015156108b457fe5b90600052602060002090600202016001015411156108f3576002818154811015156108db57fe5b90600052602060002090600202016001015491508092505b8080600101915050610897565b505090565b6000809054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff163373ffffffffffffffffffffffffffffffffffffffff161415156109ef576040517f08c379a00000000000000000000000000000000000000000000000000000000081526004018080602001828103825260288152602001807f4f6e6c79206368616972706572736f6e2063616e20676976652072696768742081526020017f746f20766f74652e00000000000000000000000000000000000000000000000081525060400191505060405180910390fd5b600160008273ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002060010160009054906101000a900460ff16151515610ab4576040517f08c379a00000000000000000000000000000000000000000000000000000000081526004018080602001828103825260188152602001807f54686520766f74657220616c726561647920766f7465642e000000000000000081525060200191505060405180910390fd5b6000600160008373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002060000154141515610b0557600080fd5b60018060008373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020019081526020016000206000018190555050565b60016020528060005260406000206000915090508060000154908060010160009054906101000a900460ff16908060010160019054906101000a900473ffffffffffffffffffffffffffffffffffffffff16908060020154905084565b60006002610bb861088a565b815481101515610bc457fe5b9060005260206000209060020201600001549050905600a165627a7a7230582045951e388fb2f21771f9e7dbe4bb8c7e91c1976b998e3849f1e3d84e76f316ae0029'

runtimeVotingContract = '60806040526004361061008e576000357c0100000000000000000000000000000000000000000000000000000000900463ffffffff1680630121b93f14610093578063013cf08b146100c05780632e4176cf146101105780635c19a95c14610167578063609ff1bd146101aa5780639e7b8d61146101d5578063a3ec138d14610218578063e2ba53f0146102b4575b600080fd5b34801561009f57600080fd5b506100be600480360381019080803590602001909291905050506102e7565b005b3480156100cc57600080fd5b506100eb6004803603810190808035906020019092919050505061040d565b6040518083600019166000191681526020018281526020019250505060405180910390f35b34801561011c57600080fd5b50610125610440565b604051808273ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200191505060405180910390f35b34801561017357600080fd5b506101a8600480360381019080803573ffffffffffffffffffffffffffffffffffffffff169060200190929190505050610465565b005b3480156101b657600080fd5b506101bf61088a565b6040518082815260200191505060405180910390f35b3480156101e157600080fd5b50610216600480360381019080803573ffffffffffffffffffffffffffffffffffffffff169060200190929190505050610905565b005b34801561022457600080fd5b50610259600480360381019080803573ffffffffffffffffffffffffffffffffffffffff169060200190929190505050610b4f565b60405180858152602001841515151581526020018373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200182815260200194505050505060405180910390f35b3480156102c057600080fd5b506102c9610bac565b60405180826000191660001916815260200191505060405180910390f35b6000600160003373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002090508060010160009054906101000a900460ff161515156103b1576040517f08c379a000000000000000000000000000000000000000000000000000000000815260040180806020018281038252600e8152602001807f416c726561647920766f7465642e00000000000000000000000000000000000081525060200191505060405180910390fd5b60018160010160006101000a81548160ff02191690831515021790555081816002018190555080600001546002838154811015156103eb57fe5b9060005260206000209060020201600101600082825401925050819055505050565b60028181548110151561041c57fe5b90600052602060002090600202016000915090508060000154908060010154905082565b6000809054906101000a900473ffffffffffffffffffffffffffffffffffffffff1681565b600080600160003373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002091508160010160009054906101000a900460ff16151515610530576040517f08c379a00000000000000000000000000000000000000000000000000000000081526004018080602001828103825260128152602001807f596f7520616c726561647920766f7465642e000000000000000000000000000081525060200191505060405180910390fd5b3373ffffffffffffffffffffffffffffffffffffffff168373ffffffffffffffffffffffffffffffffffffffff16141515156105d4576040517f08c379a000000000000000000000000000000000000000000000000000000000815260040180806020018281038252601e8152602001807f53656c662d64656c65676174696f6e20697320646973616c6c6f7765642e000081525060200191505060405180910390fd5b5b600073ffffffffffffffffffffffffffffffffffffffff16600160008573ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002060010160019054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1614151561077b57600160008473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002060010160019054906101000a900473ffffffffffffffffffffffffffffffffffffffff1692503373ffffffffffffffffffffffffffffffffffffffff168373ffffffffffffffffffffffffffffffffffffffff1614151515610776576040517f08c379a00000000000000000000000000000000000000000000000000000000081526004018080602001828103825260198152602001807f466f756e64206c6f6f7020696e2064656c65676174696f6e2e0000000000000081525060200191505060405180910390fd5b6105d5565b60018260010160006101000a81548160ff021916908315150217905550828260010160016101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff160217905550600160008473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002090508060010160009054906101000a900460ff161561086e5781600001546002826002015481548110151561084b57fe5b906000526020600020906002020160010160008282540192505081905550610885565b816000015481600001600082825401925050819055505b505050565b6000806000809150600090505b60028054905081101561090057816002828154811015156108b457fe5b90600052602060002090600202016001015411156108f3576002818154811015156108db57fe5b90600052602060002090600202016001015491508092505b8080600101915050610897565b505090565b6000809054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff163373ffffffffffffffffffffffffffffffffffffffff161415156109ef576040517f08c379a00000000000000000000000000000000000000000000000000000000081526004018080602001828103825260288152602001807f4f6e6c79206368616972706572736f6e2063616e20676976652072696768742081526020017f746f20766f74652e00000000000000000000000000000000000000000000000081525060400191505060405180910390fd5b600160008273ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002060010160009054906101000a900460ff16151515610ab4576040517f08c379a00000000000000000000000000000000000000000000000000000000081526004018080602001828103825260188152602001807f54686520766f74657220616c726561647920766f7465642e000000000000000081525060200191505060405180910390fd5b6000600160008373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002060000154141515610b0557600080fd5b60018060008373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020019081526020016000206000018190555050565b60016020528060005260406000206000915090508060000154908060010160009054906101000a900460ff16908060010160019054906101000a900473ffffffffffffffffffffffffffffffffffffffff16908060020154905084565b60006002610bb861088a565b815481101515610bc457fe5b9060005260206000209060020201600001549050905600a165627a7a7230582045951e388fb2f21771f9e7dbe4bb8c7e91c1976b998e3849f1e3d84e76f316ae0029'

votingContractABI = [
	{
		"constant": false,
		"inputs": [
			{
				"name": "proposal",
				"type": "uint256"
			}
		],
		"name": "vote",
		"outputs": [],
		"payable": false,
		"stateMutability": "nonpayable",
		"type": "function"
	},
	{
		"constant": true,
		"inputs": [
			{
				"name": "",
				"type": "uint256"
			}
		],
		"name": "proposals",
		"outputs": [
			{
				"name": "name",
				"type": "bytes32"
			},
			{
				"name": "voteCount",
				"type": "uint256"
			}
		],
		"payable": false,
		"stateMutability": "view",
		"type": "function"
	},
	{
		"constant": true,
		"inputs": [],
		"name": "chairperson",
		"outputs": [
			{
				"name": "",
				"type": "address"
			}
		],
		"payable": false,
		"stateMutability": "view",
		"type": "function"
	},
	{
		"constant": false,
		"inputs": [
			{
				"name": "to",
				"type": "address"
			}
		],
		"name": "delegate",
		"outputs": [],
		"payable": false,
		"stateMutability": "nonpayable",
		"type": "function"
	},
	{
		"constant": true,
		"inputs": [],
		"name": "winningProposal",
		"outputs": [
			{
				"name": "winningProposal_",
				"type": "uint256"
			}
		],
		"payable": false,
		"stateMutability": "view",
		"type": "function"
	},
	{
		"constant": false,
		"inputs": [
			{
				"name": "voter",
				"type": "address"
			}
		],
		"name": "giveRightToVote",
		"outputs": [],
		"payable": false,
		"stateMutability": "nonpayable",
		"type": "function"
	},
	{
		"constant": true,
		"inputs": [
			{
				"name": "",
				"type": "address"
			}
		],
		"name": "voters",
		"outputs": [
			{
				"name": "weight",
				"type": "uint256"
			},
			{
				"name": "voted",
				"type": "bool"
			},
			{
				"name": "delegate",
				"type": "address"
			},
			{
				"name": "vote",
				"type": "uint256"
			}
		],
		"payable": false,
		"stateMutability": "view",
		"type": "function"
	},
	{
		"constant": true,
		"inputs": [],
		"name": "winnerName",
		"outputs": [
			{
				"name": "winnerName_",
				"type": "bytes32"
			}
		],
		"payable": false,
		"stateMutability": "view",
		"type": "function"
	},
	{
		"inputs": [
			{
				"name": "proposalNames",
				"type": "bytes32[]"
			}
		],
		"payable": false,
		"stateMutability": "nonpayable",
		"type": "constructor"
	}
]

module.exports.compiledVotingContract = compiledVotingContract
module.exports.runtimeVotingContract = runtimeVotingContract
module.exports.votingContractABI = votingContractABI
