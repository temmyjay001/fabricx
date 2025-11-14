package main

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

// AssetTransfer provides functions for managing assets
type AssetTransfer struct {
	contractapi.Contract
}

// Asset describes basic details of an asset
type Asset struct {
	ID             string    `json:"id"`
	Owner          string    `json:"owner"`
	Value          int       `json:"value"`
	Color          string    `json:"color"`
	Size           int       `json:"size"`
	AppraisedValue int       `json:"appraisedValue"`
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
}

// AssetHistory represents the history of an asset
type AssetHistory struct {
	TxID      string    `json:"txId"`
	Timestamp time.Time `json:"timestamp"`
	Asset     Asset     `json:"asset"`
	IsDelete  bool      `json:"isDelete"`
}

// InitLedger adds a base set of assets to the ledger
func (a *AssetTransfer) InitLedger(ctx contractapi.TransactionContextInterface) error {
	assets := []Asset{
		{ID: "asset1", Owner: "Alice", Value: 300, Color: "blue", Size: 5, AppraisedValue: 300, CreatedAt: time.Now(), UpdatedAt: time.Now()},
		{ID: "asset2", Owner: "Bob", Value: 400, Color: "red", Size: 5, AppraisedValue: 400, CreatedAt: time.Now(), UpdatedAt: time.Now()},
		{ID: "asset3", Owner: "Charlie", Value: 500, Color: "green", Size: 10, AppraisedValue: 500, CreatedAt: time.Now(), UpdatedAt: time.Now()},
		{ID: "asset4", Owner: "Diana", Value: 600, Color: "yellow", Size: 10, AppraisedValue: 600, CreatedAt: time.Now(), UpdatedAt: time.Now()},
		{ID: "asset5", Owner: "Eve", Value: 700, Color: "black", Size: 15, AppraisedValue: 700, CreatedAt: time.Now(), UpdatedAt: time.Now()},
	}

	for _, asset := range assets {
		assetJSON, err := json.Marshal(asset)
		if err != nil {
			return err
		}

		err = ctx.GetStub().PutState(asset.ID, assetJSON)
		if err != nil {
			return fmt.Errorf("failed to put asset to world state: %v", err)
		}

		log.Printf("Asset %s initialized", asset.ID)
	}

	return nil
}

// CreateAsset issues a new asset to the world state
func (a *AssetTransfer) CreateAsset(ctx contractapi.TransactionContextInterface, id string, owner string, value int, color string, size int, appraisedValue int) error {
	exists, err := a.AssetExists(ctx, id)
	if err != nil {
		return err
	}
	if exists {
		return fmt.Errorf("asset %s already exists", id)
	}

	asset := Asset{
		ID:             id,
		Owner:          owner,
		Value:          value,
		Color:          color,
		Size:           size,
		AppraisedValue: appraisedValue,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	assetJSON, err := json.Marshal(asset)
	if err != nil {
		return err
	}

	err = ctx.GetStub().PutState(id, assetJSON)
	if err != nil {
		return err
	}

	// Emit creation event
	err = ctx.GetStub().SetEvent("AssetCreated", assetJSON)
	if err != nil {
		return fmt.Errorf("failed to set event: %v", err)
	}

	log.Printf("Asset %s created by %s", id, owner)
	return nil
}

// ReadAsset returns the asset stored in the world state with given id
func (a *AssetTransfer) ReadAsset(ctx contractapi.TransactionContextInterface, id string) (*Asset, error) {
	assetJSON, err := ctx.GetStub().GetState(id)
	if err != nil {
		return nil, fmt.Errorf("failed to read from world state: %v", err)
	}
	if assetJSON == nil {
		return nil, fmt.Errorf("asset %s does not exist", id)
	}

	var asset Asset
	err = json.Unmarshal(assetJSON, &asset)
	if err != nil {
		return nil, err
	}

	return &asset, nil
}

// UpdateAsset updates an existing asset in the world state
func (a *AssetTransfer) UpdateAsset(ctx contractapi.TransactionContextInterface, id string, owner string, value int, color string, size int, appraisedValue int) error {
	exists, err := a.AssetExists(ctx, id)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("asset %s does not exist", id)
	}

	// Read existing asset to preserve creation time
	existingAsset, err := a.ReadAsset(ctx, id)
	if err != nil {
		return err
	}

	asset := Asset{
		ID:             id,
		Owner:          owner,
		Value:          value,
		Color:          color,
		Size:           size,
		AppraisedValue: appraisedValue,
		CreatedAt:      existingAsset.CreatedAt,
		UpdatedAt:      time.Now(),
	}

	assetJSON, err := json.Marshal(asset)
	if err != nil {
		return err
	}

	err = ctx.GetStub().PutState(id, assetJSON)
	if err != nil {
		return err
	}

	// Emit update event
	err = ctx.GetStub().SetEvent("AssetUpdated", assetJSON)
	if err != nil {
		return fmt.Errorf("failed to set event: %v", err)
	}

	log.Printf("Asset %s updated", id)
	return nil
}

// DeleteAsset deletes an asset from the world state
func (a *AssetTransfer) DeleteAsset(ctx contractapi.TransactionContextInterface, id string) error {
	exists, err := a.AssetExists(ctx, id)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("asset %s does not exist", id)
	}

	err = ctx.GetStub().DelState(id)
	if err != nil {
		return fmt.Errorf("failed to delete asset: %v", err)
	}

	// Emit deletion event
	eventPayload := map[string]string{"id": id}
	eventJSON, _ := json.Marshal(eventPayload)
	err = ctx.GetStub().SetEvent("AssetDeleted", eventJSON)
	if err != nil {
		return fmt.Errorf("failed to set event: %v", err)
	}

	log.Printf("Asset %s deleted", id)
	return nil
}

// AssetExists returns true when asset with given ID exists
func (a *AssetTransfer) AssetExists(ctx contractapi.TransactionContextInterface, id string) (bool, error) {
	assetJSON, err := ctx.GetStub().GetState(id)
	if err != nil {
		return false, fmt.Errorf("failed to read from world state: %v", err)
	}

	return assetJSON != nil, nil
}

// TransferAsset updates the owner field of asset with given id
func (a *AssetTransfer) TransferAsset(ctx contractapi.TransactionContextInterface, id string, newOwner string) error {
	asset, err := a.ReadAsset(ctx, id)
	if err != nil {
		return err
	}

	oldOwner := asset.Owner
	asset.Owner = newOwner
	asset.UpdatedAt = time.Now()

	assetJSON, err := json.Marshal(asset)
	if err != nil {
		return err
	}

	err = ctx.GetStub().PutState(id, assetJSON)
	if err != nil {
		return err
	}

	// Emit transfer event
	transferEvent := map[string]string{
		"assetId":  id,
		"from":     oldOwner,
		"to":       newOwner,
		"txId":     ctx.GetStub().GetTxID(),
		"timestamp": time.Now().Format(time.RFC3339),
	}
	transferEventJSON, _ := json.Marshal(transferEvent)
	err = ctx.GetStub().SetEvent("AssetTransferred", transferEventJSON)
	if err != nil {
		return fmt.Errorf("failed to set event: %v", err)
	}

	log.Printf("Asset %s transferred from %s to %s", id, oldOwner, newOwner)
	return nil
}

// GetAllAssets returns all assets found in world state
func (a *AssetTransfer) GetAllAssets(ctx contractapi.TransactionContextInterface) ([]*Asset, error) {
	resultsIterator, err := ctx.GetStub().GetStateByRange("", "")
	if err != nil {
		return nil, err
	}
	defer resultsIterator.Close()

	var assets []*Asset
	for resultsIterator.HasNext() {
		queryResponse, err := resultsIterator.Next()
		if err != nil {
			return nil, err
		}

		var asset Asset
		err = json.Unmarshal(queryResponse.Value, &asset)
		if err != nil {
			return nil, err
		}
		assets = append(assets, &asset)
	}

	return assets, nil
}

// GetAssetsByOwner returns all assets owned by a specific owner
func (a *AssetTransfer) GetAssetsByOwner(ctx contractapi.TransactionContextInterface, owner string) ([]*Asset, error) {
	queryString := fmt.Sprintf(`{"selector":{"owner":"%s"}}`, owner)

	resultsIterator, err := ctx.GetStub().GetQueryResult(queryString)
	if err != nil {
		return nil, err
	}
	defer resultsIterator.Close()

	var assets []*Asset
	for resultsIterator.HasNext() {
		queryResponse, err := resultsIterator.Next()
		if err != nil {
			return nil, err
		}

		var asset Asset
		err = json.Unmarshal(queryResponse.Value, &asset)
		if err != nil {
			return nil, err
		}
		assets = append(assets, &asset)
	}

	return assets, nil
}

// GetAssetHistory returns the history of an asset
func (a *AssetTransfer) GetAssetHistory(ctx contractapi.TransactionContextInterface, id string) ([]AssetHistory, error) {
	resultsIterator, err := ctx.GetStub().GetHistoryForKey(id)
	if err != nil {
		return nil, err
	}
	defer resultsIterator.Close()

	var history []AssetHistory
	for resultsIterator.HasNext() {
		response, err := resultsIterator.Next()
		if err != nil {
			return nil, err
		}

		var asset Asset
		if len(response.Value) > 0 {
			err = json.Unmarshal(response.Value, &asset)
			if err != nil {
				return nil, err
			}
		}

		record := AssetHistory{
			TxID:      response.TxId,
			Timestamp: time.Unix(response.Timestamp.Seconds, int64(response.Timestamp.Nanos)),
			Asset:     asset,
			IsDelete:  response.IsDelete,
		}
		history = append(history, record)
	}

	return history, nil
}

func main() {
	chaincode, err := contractapi.NewChaincode(&AssetTransfer{})
	if err != nil {
		log.Panicf("Error creating asset-transfer chaincode: %v", err)
	}

	if err := chaincode.Start(); err != nil {
		log.Panicf("Error starting asset-transfer chaincode: %v", err)
	}
}