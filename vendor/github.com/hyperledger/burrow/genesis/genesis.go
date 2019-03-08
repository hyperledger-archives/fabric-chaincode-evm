// Copyright 2017 Monax Industries Limited
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package genesis

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/hyperledger/burrow/binary"

	"github.com/hyperledger/burrow/acm"
	"github.com/hyperledger/burrow/acm/validator"
	"github.com/hyperledger/burrow/crypto"
	"github.com/hyperledger/burrow/permission"
)

// How many bytes to take from the front of the GenesisDoc hash to append
// to the ChainName to form the ChainID. The idea is to avoid some classes
// of replay attack between chains with the same name.
const ShortHashSuffixBytes = 3

//------------------------------------------------------------
// core types for a genesis definition

type BasicAccount struct {
	// Address  is convenient to have in file for reference, but otherwise ignored since derived from PublicKey
	Address   crypto.Address
	PublicKey crypto.PublicKey
	Amount    uint64
}

type Account struct {
	BasicAccount
	Name        string
	Permissions permission.AccountPermissions
}

type Validator struct {
	BasicAccount
	NodeAddress *crypto.Address `json:",omitempty" toml:",omitempty" yaml:",omitempty"`
	Name        string
	UnbondTo    []BasicAccount
}

//------------------------------------------------------------
// GenesisDoc is stored in the state database

const DefaultProposalThreshold uint64 = 3

type params struct {
	ProposalThreshold uint64
}

type GenesisDoc struct {
	GenesisTime       time.Time
	ChainName         string
	AppHash           binary.HexBytes `json:",omitempty" toml:",omitempty"`
	Params            params          `json:",omitempty" toml:",omitempty"`
	Salt              []byte          `json:",omitempty" toml:",omitempty"`
	GlobalPermissions permission.AccountPermissions
	Accounts          []Account
	Validators        []Validator
	// memo
	hash []byte
}

func (genesisDoc *GenesisDoc) JSONString() string {
	bs, err := genesisDoc.JSONBytes()
	if err != nil {
		return fmt.Sprintf("error marshalling GenesisDoc: %v", err)
	}
	return string(bs)
}

// JSONBytes returns the JSON canonical bytes for a given GenesisDoc or an error.
func (genesisDoc *GenesisDoc) JSONBytes() ([]byte, error) {
	// Just in case
	genesisDoc.GenesisTime = genesisDoc.GenesisTime.UTC()
	return json.MarshalIndent(genesisDoc, "", "\t")
}

func (genesisDoc *GenesisDoc) Hash() []byte {
	if genesisDoc.hash != nil {
		return genesisDoc.hash
	}

	genesisDocBytes, err := genesisDoc.JSONBytes()
	if err != nil {
		panic(fmt.Errorf("could not create hash of GenesisDoc: %v", err))
	}
	hasher := sha256.New()
	hasher.Write(genesisDocBytes)
	return hasher.Sum(nil)
}

func (genesisDoc *GenesisDoc) ShortHash() []byte {
	return genesisDoc.Hash()[:ShortHashSuffixBytes]
}

func (genesisDoc *GenesisDoc) ChainID() string {
	return fmt.Sprintf("%s-%X", genesisDoc.ChainName, genesisDoc.ShortHash())
}

//------------------------------------------------------------
// Make genesis state from file

func GenesisDocFromJSON(jsonBlob []byte) (*GenesisDoc, error) {
	genDoc := new(GenesisDoc)
	err := json.Unmarshal(jsonBlob, genDoc)
	if err != nil {
		return nil, fmt.Errorf("couldn't read GenesisDoc: %v", err)
	}
	if len(genDoc.AppHash) != 0 {
		genDoc.hash = genDoc.AppHash
	}

	return genDoc, nil

}

//------------------------------------------------------------
// Account methods

func GenesisAccountFromAccount(name string, account *acm.Account) Account {
	return Account{
		Name:        name,
		Permissions: account.Permissions,
		BasicAccount: BasicAccount{
			Address: account.Address,
			Amount:  account.Balance,
		},
	}
}

// Clone clones the genesis account
func (genesisAccount *Account) Clone() Account {
	// clone the account permissions
	return Account{
		BasicAccount: BasicAccount{
			Address: genesisAccount.Address,
			Amount:  genesisAccount.Amount,
		},
		Name:        genesisAccount.Name,
		Permissions: genesisAccount.Permissions.Clone(),
	}
}

func (genesisAccount *Account) AcmAccount() *acm.Account {
	return &acm.Account{
		Address:     genesisAccount.Address,
		PublicKey:   genesisAccount.PublicKey,
		Balance:     genesisAccount.Amount,
		Permissions: genesisAccount.Permissions,
	}
}

//------------------------------------------------------------
// Validator methods

func (gv *Validator) Validator() validator.Validator {
	address := gv.PublicKey.GetAddress()
	return validator.Validator{
		Address:   &address,
		PublicKey: gv.PublicKey,
		Power:     uint64(gv.Amount),
	}
}

// Clone clones the genesis validator
func (gv *Validator) Clone() Validator {
	// clone the addresses to unbond to
	unbondToClone := make([]BasicAccount, len(gv.UnbondTo))
	for i, basicAccount := range gv.UnbondTo {
		unbondToClone[i] = basicAccount.Clone()
	}
	return Validator{
		BasicAccount: BasicAccount{
			PublicKey: gv.PublicKey,
			Amount:    gv.Amount,
		},
		Name:        gv.Name,
		UnbondTo:    unbondToClone,
		NodeAddress: gv.NodeAddress,
	}
}

//------------------------------------------------------------
// BasicAccount methods

// Clone clones the basic account
func (basicAccount *BasicAccount) Clone() BasicAccount {
	return BasicAccount{
		Address: basicAccount.Address,
		Amount:  basicAccount.Amount,
	}
}

// MakeGenesisDocFromAccounts takes a chainName and a slice of pointers to Account,
// and a slice of pointers to Validator to construct a GenesisDoc, or returns an error on
// failure.  In particular MakeGenesisDocFromAccount uses the local time as a
// timestamp for the GenesisDoc.
func MakeGenesisDocFromAccounts(chainName string, salt []byte, genesisTime time.Time, accounts map[string]*acm.Account,
	validators map[string]*validator.Validator) *GenesisDoc {

	// Establish deterministic order of accounts by name so we obtain identical GenesisDoc
	// from identical input
	names := make([]string, 0, len(accounts))
	for name := range accounts {
		names = append(names, name)
	}
	sort.Strings(names)
	// copy slice of pointers to accounts into slice of accounts
	genesisAccounts := make([]Account, 0, len(accounts))
	for _, name := range names {
		genesisAccounts = append(genesisAccounts, GenesisAccountFromAccount(name, accounts[name]))
	}
	// Sigh...
	names = names[:0]
	for name := range validators {
		names = append(names, name)
	}
	sort.Strings(names)
	// copy slice of pointers to validators into slice of validators
	genesisValidators := make([]Validator, 0, len(validators))
	for _, name := range names {
		val := validators[name]
		genesisValidators = append(genesisValidators, Validator{
			Name: name,
			BasicAccount: BasicAccount{
				Address:   *val.Address,
				PublicKey: val.PublicKey,
				Amount:    val.Power,
			},
			// Simpler to just do this by convention
			UnbondTo: []BasicAccount{
				{
					Amount:  val.Power,
					Address: *val.Address,
				},
			},
		})
	}
	return &GenesisDoc{
		ChainName:         chainName,
		Salt:              salt,
		GenesisTime:       genesisTime,
		GlobalPermissions: permission.DefaultAccountPermissions.Clone(),
		Accounts:          genesisAccounts,
		Validators:        genesisValidators,
	}
}
