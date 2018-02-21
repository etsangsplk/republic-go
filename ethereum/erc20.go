package ethereum

import (
	"context"
	"crypto/rand"
	"log"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/republicprotocol/go-atom/ethereum/contracts"
)

// ERC20AtomContract ...
type ERC20AtomContract struct {
	context      context.Context
	client       Client
	auth         *bind.TransactOpts
	binding      *contracts.AtomicSwapERC20
	erc20        *contracts.ERC20
	erc20Address common.Address
	swapID       [32]byte
	chainID      int8
}

// NewERC20AtomContract returns a new NewERC20Atom instance
func NewERC20AtomContract(context context.Context, client Client, auth1 *bind.TransactOpts, address common.Address, erc20Address common.Address, data []byte) (*ERC20AtomContract, error) {
	contract, err := contracts.NewAtomicSwapERC20(address, bind.ContractBackend(client))
	if err != nil {
		return nil, err
	}

	erc20, err := contracts.NewERC20(erc20Address, bind.ContractBackend(client))
	if err != nil {
		return nil, err
	}

	var swapID [32]byte
	if data == nil {
		swapID = [32]byte{}
		_, err = rand.Read(swapID[:])
		if err != nil {
			return nil, err
		}
	} else {
		swapID, err = BytesTo32Bytes(data)
	}

	return &ERC20AtomContract{
		context:      context,
		client:       client,
		auth:         auth1,
		binding:      contract,
		erc20:        erc20,
		erc20Address: erc20Address,
		swapID:       swapID,
	}, nil
}

// Initiate starts or reciprocates an atomic swap
func (contract *ERC20AtomContract) Initiate(hash, to, from []byte, value *big.Int, expiry int64) (err error) {
	hash32, err := BytesTo32Bytes(hash)
	if err != nil {
		log.Fatalf("Expected 32 bytes: %v", err)
	}
	toAddress := common.BytesToAddress(to)

	// Approve ERC20 to atomic-swap contract
	tx, err := contract.erc20.Approve(contract.auth, contract.erc20Address, value)
	if err != nil {
		return err
	}
	_, err = PatchedWaitMined(contract.context, contract.client, tx)
	if err != nil {
		return err
	}

	// Call atomic-swap contract
	tx, err = contract.binding.Open(contract.auth, contract.swapID, value, contract.erc20Address, toAddress, hash32, big.NewInt(expiry))
	if err != nil {
		return err
	}
	_, err = PatchedWaitMined(contract.context, contract.client, tx)
	return err
}

// Read returns details about an atomic swap
func (contract *ERC20AtomContract) Read() (hash, to, from []byte, value *big.Int, expiry int64, err error) {
	ret, err := contract.binding.Check(&bind.CallOpts{}, contract.swapID)
	return ret.SecretLock[:],
		ret.WithdrawTrader.Bytes(),
		ret.Erc20ContractAddress.Bytes(),
		ret.Erc20Value,
		ret.Timelock.Int64(),
		err
}

// ReadSecret returns the secret of an atomic swap if it's available
func (contract *ERC20AtomContract) ReadSecret() (secret []byte, err error) {
	return contract.binding.CheckSecretKey(&bind.CallOpts{}, contract.swapID)
}

// Redeem ...
func (contract *ERC20AtomContract) Redeem(secret []byte) error {
	tx, err := contract.binding.Close(contract.auth, contract.swapID, secret)
	if err != nil {
		return err
	}
	_, err = PatchedWaitMined(contract.context, contract.client, tx)
	return err
}

// Refund will return the funds of an atomic swap, if the expiry period has passed
func (contract *ERC20AtomContract) Refund() error {
	tx, err := contract.binding.Expire(contract.auth, contract.swapID)
	if err != nil {
		return err
	}
	_, err = PatchedWaitMined(contract.context, contract.client, tx)
	return err
}

// GetData returns the data required for another party to participate in an atomic swap
func (contract *ERC20AtomContract) GetData() []byte {
	return contract.swapID[:]
}

/*

// Open opens an Atomic swap for a given match ID, with a address authorised to withdraw the amount after revealing the secret
func (connection ERC20Connection) Open(_swapID [32]byte, ethAddr common.Address, secretHash [32]byte, amountInWei *big.Int) (*types.Transaction, error) {
	return connection.contract.Open(connection.auth, _swapID, amountInWei, ethAddr, ethAddr, secretHash)
}

// Close closes an Atomic swap by revealing the secret. The locked value is sent to the address supplied to Open
func (connection ERC20Connection) Close(_swapID [32]byte, _secretKey []byte) (*types.Transaction, error) {
	return connection.contract.Close(connection.auth, _swapID, _secretKey)
}

// Check returns details about an open Atomic Swap
func (connection ERC20Connection) Check(id [32]byte) (struct {
	TimeRemaining        *big.Int
	Erc20Value           *big.Int
	Erc20ContractAddress common.Address
	WithdrawTrader       common.Address
	SecretLock           [32]byte
}, error) {
	return connection.contract.Check(&bind.CallOpts{}, id)
}

// Expire expires an Atomic Swap, provided that the required time has passed
func (connection ERC20Connection) Expire(_swapID [32]byte) (*types.Transaction, error) {
	return connection.contract.Expire(connection.auth, _swapID)
}

// Validate (not implemented) checks that there is a valid open Atomic Swap for a given _swapID
func (connection ERC20Connection) Validate() {
}

// RetrieveSecretKey retrieves the secret key from an Atomic Swap, after it has been revealed
func (connection ERC20Connection) RetrieveSecretKey(_swapID [32]byte) ([]byte, error) {
	return connection.contract.CheckSecretKey(&bind.CallOpts{}, _swapID)
}



func existingERC20(connection bind.ContractBackend, address common.Address) *contracts.AtomicSwapERC20 {
	contract, err := contracts.NewAtomicSwapERC20(address, connection)
	if err != nil {
		log.Fatalf("%v", err)
	}
	return contract
}

*/
