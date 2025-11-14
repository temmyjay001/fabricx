// templates/erc20/go/main.go
package main

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

// ERC20Token provides token management functions
type ERC20Token struct {
	contractapi.Contract
}

// TokenMetadata holds token information
type TokenMetadata struct {
	Name        string `json:"name"`
	Symbol      string `json:"symbol"`
	Decimals    uint8  `json:"decimals"`
	TotalSupply uint64 `json:"totalSupply"`
	Owner       string `json:"owner"`
}

// TransferEvent represents a token transfer
type TransferEvent struct {
	From   string `json:"from"`
	To     string `json:"to"`
	Amount uint64 `json:"amount"`
}

// ApprovalEvent represents an approval
type ApprovalEvent struct {
	Owner   string `json:"owner"`
	Spender string `json:"spender"`
	Amount  uint64 `json:"amount"`
}

// Initialize creates a new token
func (t *ERC20Token) Initialize(ctx contractapi.TransactionContextInterface, name string, symbol string, decimals uint8, initialSupply uint64) error {
	// Check if already initialized
	exists, err := t.TokenExists(ctx)
	if err != nil {
		return err
	}
	if exists {
		return fmt.Errorf("token already initialized")
	}

	clientID, err := ctx.GetClientIdentity().GetID()
	if err != nil {
		return fmt.Errorf("failed to get client identity: %v", err)
	}

	metadata := TokenMetadata{
		Name:        name,
		Symbol:      symbol,
		Decimals:    decimals,
		TotalSupply: initialSupply,
		Owner:       clientID,
	}

	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		return err
	}

	err = ctx.GetStub().PutState("metadata", metadataJSON)
	if err != nil {
		return err
	}

	// Mint initial supply to owner
	return t.mint(ctx, clientID, initialSupply)
}

// TokenExists checks if token is initialized
func (t *ERC20Token) TokenExists(ctx contractapi.TransactionContextInterface) (bool, error) {
	metadataJSON, err := ctx.GetStub().GetState("metadata")
	if err != nil {
		return false, err
	}
	return metadataJSON != nil, nil
}

// Name returns the token name
func (t *ERC20Token) Name(ctx contractapi.TransactionContextInterface) (string, error) {
	metadata, err := t.getMetadata(ctx)
	if err != nil {
		return "", err
	}
	return metadata.Name, nil
}

// Symbol returns the token symbol
func (t *ERC20Token) Symbol(ctx contractapi.TransactionContextInterface) (string, error) {
	metadata, err := t.getMetadata(ctx)
	if err != nil {
		return "", err
	}
	return metadata.Symbol, nil
}

// Decimals returns the token decimals
func (t *ERC20Token) Decimals(ctx contractapi.TransactionContextInterface) (uint8, error) {
	metadata, err := t.getMetadata(ctx)
	if err != nil {
		return 0, err
	}
	return metadata.Decimals, nil
}

// TotalSupply returns the total token supply
func (t *ERC20Token) TotalSupply(ctx contractapi.TransactionContextInterface) (uint64, error) {
	metadata, err := t.getMetadata(ctx)
	if err != nil {
		return 0, err
	}
	return metadata.TotalSupply, nil
}

// BalanceOf returns the balance of an account
func (t *ERC20Token) BalanceOf(ctx contractapi.TransactionContextInterface, account string) (uint64, error) {
	balanceKey := fmt.Sprintf("balance_%s", account)
	balanceJSON, err := ctx.GetStub().GetState(balanceKey)
	if err != nil {
		return 0, err
	}

	if balanceJSON == nil {
		return 0, nil
	}

	var balance uint64
	err = json.Unmarshal(balanceJSON, &balance)
	if err != nil {
		return 0, err
	}

	return balance, nil
}

// Transfer transfers tokens from caller to recipient
func (t *ERC20Token) Transfer(ctx contractapi.TransactionContextInterface, to string, amount uint64) error {
	clientID, err := ctx.GetClientIdentity().GetID()
	if err != nil {
		return fmt.Errorf("failed to get client identity: %v", err)
	}

	return t.transferHelper(ctx, clientID, to, amount)
}

// TransferFrom transfers tokens from one account to another using allowance
func (t *ERC20Token) TransferFrom(ctx contractapi.TransactionContextInterface, from string, to string, amount uint64) error {
	clientID, err := ctx.GetClientIdentity().GetID()
	if err != nil {
		return fmt.Errorf("failed to get client identity: %v", err)
	}

	// Check allowance
	allowance, err := t.Allowance(ctx, from, clientID)
	if err != nil {
		return err
	}

	if allowance < amount {
		return fmt.Errorf("insufficient allowance: have %d, need %d", allowance, amount)
	}

	// Transfer tokens
	err = t.transferHelper(ctx, from, to, amount)
	if err != nil {
		return err
	}

	// Update allowance
	newAllowance := allowance - amount
	return t.setAllowance(ctx, from, clientID, newAllowance)
}

// Approve approves spender to spend amount on behalf of caller
func (t *ERC20Token) Approve(ctx contractapi.TransactionContextInterface, spender string, amount uint64) error {
	clientID, err := ctx.GetClientIdentity().GetID()
	if err != nil {
		return fmt.Errorf("failed to get client identity: %v", err)
	}

	err = t.setAllowance(ctx, clientID, spender, amount)
	if err != nil {
		return err
	}

	// Emit approval event
	event := ApprovalEvent{
		Owner:   clientID,
		Spender: spender,
		Amount:  amount,
	}
	eventJSON, _ := json.Marshal(event)
	ctx.GetStub().SetEvent("Approval", eventJSON)

	log.Printf("Approval: %s approved %s to spend %d", clientID, spender, amount)
	return nil
}

// Allowance returns the remaining allowance
func (t *ERC20Token) Allowance(ctx contractapi.TransactionContextInterface, owner string, spender string) (uint64, error) {
	allowanceKey := fmt.Sprintf("allowance_%s_%s", owner, spender)
	allowanceJSON, err := ctx.GetStub().GetState(allowanceKey)
	if err != nil {
		return 0, err
	}

	if allowanceJSON == nil {
		return 0, nil
	}

	var allowance uint64
	err = json.Unmarshal(allowanceJSON, &allowance)
	if err != nil {
		return 0, err
	}

	return allowance, nil
}

// Mint creates new tokens (only owner)
func (t *ERC20Token) Mint(ctx contractapi.TransactionContextInterface, to string, amount uint64) error {
	clientID, err := ctx.GetClientIdentity().GetID()
	if err != nil {
		return fmt.Errorf("failed to get client identity: %v", err)
	}

	metadata, err := t.getMetadata(ctx)
	if err != nil {
		return err
	}

	if clientID != metadata.Owner {
		return fmt.Errorf("only owner can mint tokens")
	}

	err = t.mint(ctx, to, amount)
	if err != nil {
		return err
	}

	// Update total supply
	metadata.TotalSupply += amount
	metadataJSON, _ := json.Marshal(metadata)
	return ctx.GetStub().PutState("metadata", metadataJSON)
}

// Burn destroys tokens
func (t *ERC20Token) Burn(ctx contractapi.TransactionContextInterface, amount uint64) error {
	clientID, err := ctx.GetClientIdentity().GetID()
	if err != nil {
		return fmt.Errorf("failed to get client identity: %v", err)
	}

	balance, err := t.BalanceOf(ctx, clientID)
	if err != nil {
		return err
	}

	if balance < amount {
		return fmt.Errorf("insufficient balance: have %d, need %d", balance, amount)
	}

	// Reduce balance
	newBalance := balance - amount
	err = t.setBalance(ctx, clientID, newBalance)
	if err != nil {
		return err
	}

	// Update total supply
	metadata, err := t.getMetadata(ctx)
	if err != nil {
		return err
	}

	metadata.TotalSupply -= amount
	metadataJSON, _ := json.Marshal(metadata)
	err = ctx.GetStub().PutState("metadata", metadataJSON)
	if err != nil {
		return err
	}

	// Emit transfer event to zero address
	event := TransferEvent{
		From:   clientID,
		To:     "0x0",
		Amount: amount,
	}
	eventJSON, _ := json.Marshal(event)
	ctx.GetStub().SetEvent("Transfer", eventJSON)

	log.Printf("Burned %d tokens from %s", amount, clientID)
	return nil
}

// Helper functions

func (t *ERC20Token) getMetadata(ctx contractapi.TransactionContextInterface) (*TokenMetadata, error) {
	metadataJSON, err := ctx.GetStub().GetState("metadata")
	if err != nil {
		return nil, err
	}
	if metadataJSON == nil {
		return nil, fmt.Errorf("token not initialized")
	}

	var metadata TokenMetadata
	err = json.Unmarshal(metadataJSON, &metadata)
	if err != nil {
		return nil, err
	}

	return &metadata, nil
}

func (t *ERC20Token) setBalance(ctx contractapi.TransactionContextInterface, account string, balance uint64) error {
	balanceKey := fmt.Sprintf("balance_%s", account)
	balanceJSON, err := json.Marshal(balance)
	if err != nil {
		return err
	}
	return ctx.GetStub().PutState(balanceKey, balanceJSON)
}

func (t *ERC20Token) setAllowance(ctx contractapi.TransactionContextInterface, owner string, spender string, amount uint64) error {
	allowanceKey := fmt.Sprintf("allowance_%s_%s", owner, spender)
	allowanceJSON, err := json.Marshal(amount)
	if err != nil {
		return err
	}
	return ctx.GetStub().PutState(allowanceKey, allowanceJSON)
}

func (t *ERC20Token) mint(ctx contractapi.TransactionContextInterface, to string, amount uint64) error {
	balance, err := t.BalanceOf(ctx, to)
	if err != nil {
		return err
	}

	newBalance := balance + amount
	err = t.setBalance(ctx, to, newBalance)
	if err != nil {
		return err
	}

	// Emit transfer event from zero address
	event := TransferEvent{
		From:   "0x0",
		To:     to,
		Amount: amount,
	}
	eventJSON, _ := json.Marshal(event)
	ctx.GetStub().SetEvent("Transfer", eventJSON)

	log.Printf("Minted %d tokens to %s", amount, to)
	return nil
}

func (t *ERC20Token) transferHelper(ctx contractapi.TransactionContextInterface, from string, to string, amount uint64) error {
	if amount == 0 {
		return fmt.Errorf("transfer amount must be positive")
	}

	// Get sender balance
	fromBalance, err := t.BalanceOf(ctx, from)
	if err != nil {
		return err
	}

	if fromBalance < amount {
		return fmt.Errorf("insufficient balance: have %d, need %d", fromBalance, amount)
	}

	// Get recipient balance
	toBalance, err := t.BalanceOf(ctx, to)
	if err != nil {
		return err
	}

	// Update balances
	err = t.setBalance(ctx, from, fromBalance-amount)
	if err != nil {
		return err
	}

	err = t.setBalance(ctx, to, toBalance+amount)
	if err != nil {
		return err
	}

	// Emit transfer event
	event := TransferEvent{
		From:   from,
		To:     to,
		Amount: amount,
	}
	eventJSON, _ := json.Marshal(event)
	ctx.GetStub().SetEvent("Transfer", eventJSON)

	log.Printf("Transferred %d tokens from %s to %s", amount, from, to)
	return nil
}

// ClientAccountID returns a unique identifier for the client
func (t *ERC20Token) ClientAccountID(ctx contractapi.TransactionContextInterface) (string, error) {
	clientID, err := ctx.GetClientIdentity().GetID()
	if err != nil {
		return "", fmt.Errorf("failed to get client identity: %v", err)
	}
	return clientID, nil
}

func main() {
	chaincode, err := contractapi.NewChaincode(&ERC20Token{})
	if err != nil {
		log.Panicf("Error creating erc20 chaincode: %v", err)
	}

	if err := chaincode.Start(); err != nil {
		log.Panicf("Error starting erc20 chaincode: %v", err)
	}
}
