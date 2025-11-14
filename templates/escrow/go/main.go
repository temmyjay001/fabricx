package main

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

// EscrowContract provides escrow management functions
type EscrowContract struct {
	contractapi.Contract
}

// EscrowStatus represents the state of an escrow
type EscrowStatus string

const (
	StatusPending   EscrowStatus = "PENDING"
	StatusFunded    EscrowStatus = "FUNDED"
	StatusReleased  EscrowStatus = "RELEASED"
	StatusRefunded  EscrowStatus = "REFUNDED"
	StatusDisputed  EscrowStatus = "DISPUTED"
	StatusCancelled EscrowStatus = "CANCELLED"
)

// Escrow represents an escrow agreement
type Escrow struct {
	ID              string       `json:"id"`
	Buyer           string       `json:"buyer"`
	Seller          string       `json:"seller"`
	Arbiter         string       `json:"arbiter"`
	Amount          uint64       `json:"amount"`
	Description     string       `json:"description"`
	Status          EscrowStatus `json:"status"`
	CreatedAt       time.Time    `json:"createdAt"`
	UpdatedAt       time.Time    `json:"updatedAt"`
	FundedAt        *time.Time   `json:"fundedAt,omitempty"`
	ReleasedAt      *time.Time   `json:"releasedAt,omitempty"`
	RefundedAt      *time.Time   `json:"refundedAt,omitempty"`
	DisputedAt      *time.Time   `json:"disputedAt,omitempty"`
	ReleaseDeadline *time.Time   `json:"releaseDeadline,omitempty"`
	Metadata        string       `json:"metadata,omitempty"`
}

// EscrowEvent represents an escrow state change
type EscrowEvent struct {
	EscrowID  string       `json:"escrowId"`
	Status    EscrowStatus `json:"status"`
	Actor     string       `json:"actor"`
	Timestamp time.Time    `json:"timestamp"`
	TxID      string       `json:"txId"`
}

// CreateEscrow creates a new escrow agreement
func (e *EscrowContract) CreateEscrow(ctx contractapi.TransactionContextInterface, id string, seller string, arbiter string, amount uint64, description string, releaseDeadlineDays int) error {
	// Check if escrow already exists
	exists, err := e.EscrowExists(ctx, id)
	if err != nil {
		return err
	}
	if exists {
		return fmt.Errorf("escrow %s already exists", id)
	}

	buyer, err := ctx.GetClientIdentity().GetID()
	if err != nil {
		return fmt.Errorf("failed to get client identity: %v", err)
	}

	// Validate inputs
	if amount == 0 {
		return fmt.Errorf("amount must be greater than zero")
	}
	if buyer == seller {
		return fmt.Errorf("buyer and seller cannot be the same")
	}

	now := time.Now()
	var releaseDeadline *time.Time
	if releaseDeadlineDays > 0 {
		deadline := now.AddDate(0, 0, releaseDeadlineDays)
		releaseDeadline = &deadline
	}

	escrow := Escrow{
		ID:              id,
		Buyer:           buyer,
		Seller:          seller,
		Arbiter:         arbiter,
		Amount:          amount,
		Description:     description,
		Status:          StatusPending,
		CreatedAt:       now,
		UpdatedAt:       now,
		ReleaseDeadline: releaseDeadline,
	}

	escrowJSON, err := json.Marshal(escrow)
	if err != nil {
		return err
	}

	err = ctx.GetStub().PutState(id, escrowJSON)
	if err != nil {
		return err
	}

	// Emit event
	e.emitEvent(ctx, id, StatusPending, buyer)

	log.Printf("Escrow %s created by %s", id, buyer)
	return nil
}

// FundEscrow funds an escrow (buyer only)
func (e *EscrowContract) FundEscrow(ctx contractapi.TransactionContextInterface, id string) error {
	escrow, err := e.GetEscrow(ctx, id)
	if err != nil {
		return err
	}

	clientID, err := ctx.GetClientIdentity().GetID()
	if err != nil {
		return fmt.Errorf("failed to get client identity: %v", err)
	}

	// Only buyer can fund
	if clientID != escrow.Buyer {
		return fmt.Errorf("only buyer can fund escrow")
	}

	// Check status
	if escrow.Status != StatusPending {
		return fmt.Errorf("escrow is not in PENDING status")
	}

	// Update escrow
	now := time.Now()
	escrow.Status = StatusFunded
	escrow.UpdatedAt = now
	escrow.FundedAt = &now

	err = e.updateEscrow(ctx, escrow)
	if err != nil {
		return err
	}

	// Emit event
	e.emitEvent(ctx, id, StatusFunded, clientID)

	log.Printf("Escrow %s funded by %s", id, clientID)
	return nil
}

// ReleaseEscrow releases funds to seller (buyer only)
func (e *EscrowContract) ReleaseEscrow(ctx contractapi.TransactionContextInterface, id string) error {
	escrow, err := e.GetEscrow(ctx, id)
	if err != nil {
		return err
	}

	clientID, err := ctx.GetClientIdentity().GetID()
	if err != nil {
		return fmt.Errorf("failed to get client identity: %v", err)
	}

	// Only buyer can release
	if clientID != escrow.Buyer {
		return fmt.Errorf("only buyer can release escrow")
	}

	// Check status
	if escrow.Status != StatusFunded {
		return fmt.Errorf("escrow is not in FUNDED status")
	}

	// Update escrow
	now := time.Now()
	escrow.Status = StatusReleased
	escrow.UpdatedAt = now
	escrow.ReleasedAt = &now

	err = e.updateEscrow(ctx, escrow)
	if err != nil {
		return err
	}

	// Emit event
	e.emitEvent(ctx, id, StatusReleased, clientID)

	log.Printf("Escrow %s released by %s to %s", id, clientID, escrow.Seller)
	return nil
}

// RefundEscrow refunds to buyer (seller only or arbiter)
func (e *EscrowContract) RefundEscrow(ctx contractapi.TransactionContextInterface, id string) error {
	escrow, err := e.GetEscrow(ctx, id)
	if err != nil {
		return err
	}

	clientID, err := ctx.GetClientIdentity().GetID()
	if err != nil {
		return fmt.Errorf("failed to get client identity: %v", err)
	}

	// Only seller or arbiter can refund
	if clientID != escrow.Seller && clientID != escrow.Arbiter {
		return fmt.Errorf("only seller or arbiter can refund escrow")
	}

	// Check status
	if escrow.Status != StatusFunded && escrow.Status != StatusDisputed {
		return fmt.Errorf("escrow must be FUNDED or DISPUTED to refund")
	}

	// Update escrow
	now := time.Now()
	escrow.Status = StatusRefunded
	escrow.UpdatedAt = now
	escrow.RefundedAt = &now

	err = e.updateEscrow(ctx, escrow)
	if err != nil {
		return err
	}

	// Emit event
	e.emitEvent(ctx, id, StatusRefunded, clientID)

	log.Printf("Escrow %s refunded by %s to %s", id, clientID, escrow.Buyer)
	return nil
}

// DisputeEscrow raises a dispute (buyer or seller)
func (e *EscrowContract) DisputeEscrow(ctx contractapi.TransactionContextInterface, id string) error {
	escrow, err := e.GetEscrow(ctx, id)
	if err != nil {
		return err
	}

	clientID, err := ctx.GetClientIdentity().GetID()
	if err != nil {
		return fmt.Errorf("failed to get client identity: %v", err)
	}

	// Only buyer or seller can dispute
	if clientID != escrow.Buyer && clientID != escrow.Seller {
		return fmt.Errorf("only buyer or seller can dispute escrow")
	}

	// Check status
	if escrow.Status != StatusFunded {
		return fmt.Errorf("escrow must be FUNDED to dispute")
	}

	// Update escrow
	now := time.Now()
	escrow.Status = StatusDisputed
	escrow.UpdatedAt = now
	escrow.DisputedAt = &now

	err = e.updateEscrow(ctx, escrow)
	if err != nil {
		return err
	}

	// Emit event
	e.emitEvent(ctx, id, StatusDisputed, clientID)

	log.Printf("Escrow %s disputed by %s", id, clientID)
	return nil
}

// ResolveDispute resolves a dispute (arbiter only)
func (e *EscrowContract) ResolveDispute(ctx contractapi.TransactionContextInterface, id string, releaseToSeller bool) error {
	escrow, err := e.GetEscrow(ctx, id)
	if err != nil {
		return err
	}

	clientID, err := ctx.GetClientIdentity().GetID()
	if err != nil {
		return fmt.Errorf("failed to get client identity: %v", err)
	}

	// Only arbiter can resolve
	if clientID != escrow.Arbiter {
		return fmt.Errorf("only arbiter can resolve dispute")
	}

	// Check status
	if escrow.Status != StatusDisputed {
		return fmt.Errorf("escrow must be DISPUTED to resolve")
	}

	// Update escrow based on resolution
	now := time.Now()
	if releaseToSeller {
		escrow.Status = StatusReleased
		escrow.ReleasedAt = &now
		log.Printf("Dispute resolved: funds released to seller %s", escrow.Seller)
	} else {
		escrow.Status = StatusRefunded
		escrow.RefundedAt = &now
		log.Printf("Dispute resolved: funds refunded to buyer %s", escrow.Buyer)
	}
	escrow.UpdatedAt = now

	err = e.updateEscrow(ctx, escrow)
	if err != nil {
		return err
	}

	// Emit event
	e.emitEvent(ctx, id, escrow.Status, clientID)

	return nil
}

// CancelEscrow cancels an unfunded escrow (buyer only)
func (e *EscrowContract) CancelEscrow(ctx contractapi.TransactionContextInterface, id string) error {
	escrow, err := e.GetEscrow(ctx, id)
	if err != nil {
		return err
	}

	clientID, err := ctx.GetClientIdentity().GetID()
	if err != nil {
		return fmt.Errorf("failed to get client identity: %v", err)
	}

	// Only buyer can cancel
	if clientID != escrow.Buyer {
		return fmt.Errorf("only buyer can cancel escrow")
	}

	// Check status
	if escrow.Status != StatusPending {
		return fmt.Errorf("only PENDING escrows can be cancelled")
	}

	// Update escrow
	escrow.Status = StatusCancelled
	escrow.UpdatedAt = time.Now()

	err = e.updateEscrow(ctx, escrow)
	if err != nil {
		return err
	}

	// Emit event
	e.emitEvent(ctx, id, StatusCancelled, clientID)

	log.Printf("Escrow %s cancelled by %s", id, clientID)
	return nil
}

// GetEscrow returns an escrow by ID
func (e *EscrowContract) GetEscrow(ctx contractapi.TransactionContextInterface, id string) (*Escrow, error) {
	escrowJSON, err := ctx.GetStub().GetState(id)
	if err != nil {
		return nil, fmt.Errorf("failed to read escrow: %v", err)
	}
	if escrowJSON == nil {
		return nil, fmt.Errorf("escrow %s does not exist", id)
	}

	var escrow Escrow
	err = json.Unmarshal(escrowJSON, &escrow)
	if err != nil {
		return nil, err
	}

	return &escrow, nil
}

// EscrowExists checks if an escrow exists
func (e *EscrowContract) EscrowExists(ctx contractapi.TransactionContextInterface, id string) (bool, error) {
	escrowJSON, err := ctx.GetStub().GetState(id)
	if err != nil {
		return false, fmt.Errorf("failed to read escrow: %v", err)
	}
	return escrowJSON != nil, nil
}

// GetEscrowsByBuyer returns all escrows for a buyer
func (e *EscrowContract) GetEscrowsByBuyer(ctx contractapi.TransactionContextInterface, buyer string) ([]*Escrow, error) {
	queryString := fmt.Sprintf(`{"selector":{"buyer":"%s"}}`, buyer)
	return e.queryEscrows(ctx, queryString)
}

// GetEscrowsBySeller returns all escrows for a seller
func (e *EscrowContract) GetEscrowsBySeller(ctx contractapi.TransactionContextInterface, seller string) ([]*Escrow, error) {
	queryString := fmt.Sprintf(`{"selector":{"seller":"%s"}}`, seller)
	return e.queryEscrows(ctx, queryString)
}

// GetEscrowsByStatus returns all escrows with a specific status
func (e *EscrowContract) GetEscrowsByStatus(ctx contractapi.TransactionContextInterface, status string) ([]*Escrow, error) {
	queryString := fmt.Sprintf(`{"selector":{"status":"%s"}}`, status)
	return e.queryEscrows(ctx, queryString)
}

// GetAllEscrows returns all escrows
func (e *EscrowContract) GetAllEscrows(ctx contractapi.TransactionContextInterface) ([]*Escrow, error) {
	resultsIterator, err := ctx.GetStub().GetStateByRange("", "")
	if err != nil {
		return nil, err
	}
	defer resultsIterator.Close()

	var escrows []*Escrow
	for resultsIterator.HasNext() {
		queryResponse, err := resultsIterator.Next()
		if err != nil {
			return nil, err
		}

		var escrow Escrow
		err = json.Unmarshal(queryResponse.Value, &escrow)
		if err != nil {
			return nil, err
		}
		escrows = append(escrows, &escrow)
	}

	return escrows, nil
}

// Helper functions

func (e *EscrowContract) updateEscrow(ctx contractapi.TransactionContextInterface, escrow *Escrow) error {
	escrowJSON, err := json.Marshal(escrow)
	if err != nil {
		return err
	}
	return ctx.GetStub().PutState(escrow.ID, escrowJSON)
}

func (e *EscrowContract) queryEscrows(ctx contractapi.TransactionContextInterface, queryString string) ([]*Escrow, error) {
	resultsIterator, err := ctx.GetStub().GetQueryResult(queryString)
	if err != nil {
		return nil, err
	}
	defer resultsIterator.Close()

	var escrows []*Escrow
	for resultsIterator.HasNext() {
		queryResponse, err := resultsIterator.Next()
		if err != nil {
			return nil, err
		}

		var escrow Escrow
		err = json.Unmarshal(queryResponse.Value, &escrow)
		if err != nil {
			return nil, err
		}
		escrows = append(escrows, &escrow)
	}

	return escrows, nil
}

func (e *EscrowContract) emitEvent(ctx contractapi.TransactionContextInterface, escrowID string, status EscrowStatus, actor string) {
	event := EscrowEvent{
		EscrowID:  escrowID,
		Status:    status,
		Actor:     actor,
		Timestamp: time.Now(),
		TxID:      ctx.GetStub().GetTxID(),
	}
	eventJSON, _ := json.Marshal(event)
	ctx.GetStub().SetEvent("EscrowEvent", eventJSON)
}

func main() {
	chaincode, err := contractapi.NewChaincode(&EscrowContract{})
	if err != nil {
		log.Panicf("Error creating escrow chaincode: %v", err)
	}

	if err := chaincode.Start(); err != nil {
		log.Panicf("Error starting escrow chaincode: %v", err)
	}
}
