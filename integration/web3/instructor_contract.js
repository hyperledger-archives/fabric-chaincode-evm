/*
Copyright IBM Corp. All Rights Reserved.
SPDX-License-Identifier: Apache-2.0
*/


/* This file has both the compiled contract code, the runtime code and ABI for the Instructor Contract
 * It also contains the expected event signature for Setter
  contract Instructor {
    bytes32 fName;
    uint age;
    uint salary;
    event Setter(bytes32 indexed name, uint age, uint salary);
    function setInstructor(bytes32 _fName, uint _age, uint _salary) public {
      fName = _fName;
      age = _age;
      salary = _salary;
      emit Setter(_fName,age, _salary);
    }
    function getInstructor() public constant returns (bytes32, uint, uint) {
      return (fName, age, salary);
    }
  }
*/

compiledInstructorContract = '608060405234801561001057600080fd5b50610122806100206000396000f30060806040526004361060485763ffffffff7c010000000000000000000000000000000000000000000000000000000060003504166331fb1dff8114604d5780633c1b81a514606a575b600080fd5b348015605857600080fd5b506068600435602435604435609a565b005b348015607557600080fd5b50607c60e8565b60408051938452602084019290925282820152519081900360600190f35b6000839055600182905560028190556040805183815260208101839052815185927fe920a6ca2d94687457e136223552305dbabca6f28cf9c65d18efc2193a2369b0928290030190a2505050565b6000546001546002549091925600a165627a7a723058201d0d4d51ad39993c9b2fdbc1f69d9d3429d20fd0289a9fa663c00168bb10c2e20029'

runtimeInstructorContract = '60806040526004361060485763ffffffff7c010000000000000000000000000000000000000000000000000000000060003504166331fb1dff8114604d5780633c1b81a514606a575b600080fd5b348015605857600080fd5b506068600435602435604435609a565b005b348015607557600080fd5b50607c60e8565b60408051938452602084019290925282820152519081900360600190f35b6000839055600182905560028190556040805183815260208101839052815185927fe920a6ca2d94687457e136223552305dbabca6f28cf9c65d18efc2193a2369b0928290030190a2505050565b6000546001546002549091925600a165627a7a723058201d0d4d51ad39993c9b2fdbc1f69d9d3429d20fd0289a9fa663c00168bb10c2e20029'

instructorContractABI = [
	{
		"constant": false,
		"inputs": [
			{
				"name": "_fName",
				"type": "bytes32"
			},
			{
				"name": "_age",
				"type": "uint256"
			},
			{
				"name": "_salary",
				"type": "uint256"
			}
		],
		"name": "setInstructor",
		"outputs": [],
		"payable": false,
		"stateMutability": "nonpayable",
		"type": "function"
	},
	{
		"constant": true,
		"inputs": [],
		"name": "getInstructor",
		"outputs": [
			{
				"name": "",
				"type": "bytes32"
			},
			{
				"name": "",
				"type": "uint256"
			},
			{
				"name": "",
				"type": "uint256"
			}
		],
		"payable": false,
		"stateMutability": "view",
		"type": "function"
	},
	{
		"anonymous": false,
		"inputs": [
			{
				"indexed": true,
				"name": "name",
				"type": "bytes32"
			},
			{
				"indexed": false,
				"name": "age",
				"type": "uint256"
			},
			{
				"indexed": false,
				"name": "salary",
				"type": "uint256"
			}
		],
		"name": "Setter",
		"type": "event"
	}
]

setterEventSignature = 'e920a6ca2d94687457e136223552305dbabca6f28cf9c65d18efc2193a2369b0'

module.exports.compiledInstructorContract = compiledInstructorContract
module.exports.runtimeInstructorContract = runtimeInstructorContract
module.exports.instructorContractABI = instructorContractABI
module.exports.setterEventSignature = setterEventSignature
