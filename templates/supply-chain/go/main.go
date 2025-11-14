package main

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

// SupplyChain provides supply chain tracking functions
type SupplyChain struct {
	contractapi.Contract
}

// ProductStatus represents the state of a product
type ProductStatus string

const (
	StatusManufactured ProductStatus = "MANUFACTURED"
	StatusInTransit    ProductStatus = "IN_TRANSIT"
	StatusReceived     ProductStatus = "RECEIVED"
	StatusInspected    ProductStatus = "INSPECTED"
	StatusDelivered    ProductStatus = "DELIVERED"
	StatusRecalled     ProductStatus = "RECALLED"
)

// Product represents a product in the supply chain
type Product struct {
	ID           string        `json:"id"`
	Name         string        `json:"name"`
	Description  string        `json:"description"`
	Manufacturer string        `json:"manufacturer"`
	CurrentOwner string        `json:"currentOwner"`
	Status       ProductStatus `json:"status"`
	Location     string        `json:"location"`
	Timestamp    time.Time     `json:"timestamp"`
	Metadata     string        `json:"metadata,omitempty"`
}

// ProductHistory represents a product's journey
type ProductHistory struct {
	Product   Product `json:"product"`
	TxID      string  `json:"txId"`
	Timestamp time.Time `json:"timestamp"`
	IsDelete  bool      `json:"isDelete"`
}

// Shipment represents a shipment
type Shipment struct {
	ID           string    `json:"id"`
	ProductID    string    `json:"productId"`
	From         string    `json:"from"`
	To           string    `json:"to"`
	Carrier      string    `json:"carrier"`
	StartedAt    time.Time `json:"startedAt"`
	ExpectedAt   time.Time `json:"expectedAt"`
	DeliveredAt  *time.Time `json:"deliveredAt,omitempty"`
	TrackingInfo string    `json:"trackingInfo,omitempty"`
}

// CreateProduct creates a new product
func (s *SupplyChain) CreateProduct(ctx contractapi.TransactionContextInterface, id string, name string, description string, location string, metadata string) error {
	exists, err := s.ProductExists(ctx, id)
	if err != nil {
		return err
	}
	if exists {
		return fmt.Errorf("product %s already exists", id)
	}

	manufacturer, err := ctx.GetClientIdentity().GetID()
	if err != nil {
		return fmt.Errorf("failed to get client identity: %v", err)
	}

	product := Product{
		ID:           id,
		Name:         name,
		Description:  description,
		Manufacturer: manufacturer,
		CurrentOwner: manufacturer,
		Status:       StatusManufactured,
		Location:     location,
		Timestamp:    time.Now(),
		Metadata:     metadata,
	}

	productJSON, err := json.Marshal(product)
	if err != nil {
		return err
	}

	err = ctx.GetStub().PutState(id, productJSON)
	if err != nil {
		return err
	}

	// Emit event
	s.emitProductEvent(ctx, "ProductCreated", product)

	log.Printf("Product %s created by %s", id, manufacturer)
	return nil
}

// TransferProduct transfers product ownership
func (s *SupplyChain) TransferProduct(ctx contractapi.TransactionContextInterface, id string, newOwner string, location string) error {
	product, err := s.GetProduct(ctx, id)
	if err != nil {
		return err
	}

	clientID, err := ctx.GetClientIdentity().GetID()
	if err != nil {
		return fmt.Errorf("failed to get client identity: %v", err)
	}

	// Only current owner can transfer
	if clientID != product.CurrentOwner {
		return fmt.Errorf("only current owner can transfer product")
	}

	oldOwner := product.CurrentOwner
	product.CurrentOwner = newOwner
	product.Location = location
	product.Timestamp = time.Now()

	err = s.updateProduct(ctx, product)
	if err != nil {
		return err
	}

	// Emit event
	transferEvent := map[string]interface{}{
		"productId": id,
		"from":      oldOwner,
		"to":        newOwner,
		"location":  location,
		"timestamp": product.Timestamp,
	}
	s.emitEvent(ctx, "ProductTransferred", transferEvent)

	log.Printf("Product %s transferred from %s to %s", id, oldOwner, newOwner)
	return nil
}

// UpdateStatus updates product status
func (s *SupplyChain) UpdateStatus(ctx contractapi.TransactionContextInterface, id string, status string, location string) error {
	product, err := s.GetProduct(ctx, id)
	if err != nil {
		return err
	}

	clientID, err := ctx.GetClientIdentity().GetID()
	if err != nil {
		return fmt.Errorf("failed to get client identity: %v", err)
	}

	// Only current owner can update status
	if clientID != product.CurrentOwner {
		return fmt.Errorf("only current owner can update status")
	}

	oldStatus := product.Status
	product.Status = ProductStatus(status)
	product.Location = location
	product.Timestamp = time.Now()

	err = s.updateProduct(ctx, product)
	if err != nil {
		return err
	}

	// Emit event
	statusEvent := map[string]interface{}{
		"productId": id,
		"oldStatus": oldStatus,
		"newStatus": status,
		"location":  location,
		"timestamp": product.Timestamp,
	}
	s.emitEvent(ctx, "StatusUpdated", statusEvent)

	log.Printf("Product %s status updated from %s to %s", id, oldStatus, status)
	return nil
}

// CreateShipment creates a new shipment
func (s *SupplyChain) CreateShipment(ctx contractapi.TransactionContextInterface, id string, productID string, to string, carrier string, expectedDays int, trackingInfo string) error {
	// Verify product exists
	product, err := s.GetProduct(ctx, productID)
	if err != nil {
		return err
	}

	from, err := ctx.GetClientIdentity().GetID()
	if err != nil {
		return fmt.Errorf("failed to get client identity: %v", err)
	}

	// Only current owner can create shipment
	if from != product.CurrentOwner {
		return fmt.Errorf("only current owner can create shipment")
	}

	now := time.Now()
	expectedAt := now.AddDate(0, 0, expectedDays)

	shipment := Shipment{
		ID:           id,
		ProductID:    productID,
		From:         from,
		To:           to,
		Carrier:      carrier,
		StartedAt:    now,
		ExpectedAt:   expectedAt,
		TrackingInfo: trackingInfo,
	}

	shipmentJSON, err := json.Marshal(shipment)
	if err != nil {
		return err
	}

	shipmentKey := fmt.Sprintf("shipment_%s", id)
	err = ctx.GetStub().PutState(shipmentKey, shipmentJSON)
	if err != nil {
		return err
	}

	// Update product status
	product.Status = StatusInTransit
	product.Timestamp = now
	err = s.updateProduct(ctx, product)
	if err != nil {
		return err
	}

	// Emit event
	shipmentEvent := map[string]interface{}{
		"shipmentId":  id,
		"productId":   productID,
		"from":        from,
		"to":          to,
		"carrier":     carrier,
		"expectedAt":  expectedAt,
		"trackingInfo": trackingInfo,
	}
	s.emitEvent(ctx, "ShipmentCreated", shipmentEvent)

	log.Printf("Shipment %s created for product %s", id, productID)
	return nil
}

// CompleteShipment marks a shipment as delivered
func (s *SupplyChain) CompleteShipment(ctx contractapi.TransactionContextInterface, shipmentID string, location string) error {
	shipmentKey := fmt.Sprintf("shipment_%s", shipmentID)
	shipmentJSON, err := ctx.GetStub().GetState(shipmentKey)
	if err != nil {
		return fmt.Errorf("failed to read shipment: %v", err)
	}
	if shipmentJSON == nil {
		return fmt.Errorf("shipment %s does not exist", shipmentID)
	}

	var shipment Shipment
	err = json.Unmarshal(shipmentJSON, &shipment)
	if err != nil {
		return err
	}

	clientID, err := ctx.GetClientIdentity().GetID()
	if err != nil {
		return fmt.Errorf("failed to get client identity: %v", err)
	}

	// Only recipient can complete shipment
	if clientID != shipment.To {
		return fmt.Errorf("only recipient can complete shipment")
	}

	now := time.Now()
	shipment.DeliveredAt = &now

	// Update shipment
	shipmentJSON, err = json.Marshal(shipment)
	if err != nil {
		return err
	}
	err = ctx.GetStub().PutState(shipmentKey, shipmentJSON)
	if err != nil {
		return err
	}

	// Update product
	product, err := s.GetProduct(ctx, shipment.ProductID)
	if err != nil {
		return err
	}

	product.Status = StatusReceived
	product.CurrentOwner = shipment.To
	product.Location = location
	product.Timestamp = now

	err = s.updateProduct(ctx, product)
	if err != nil {
		return err
	}

	// Emit event
	deliveryEvent := map[string]interface{}{
		"shipmentId": shipmentID,
		"productId":  shipment.ProductID,
		"deliveredAt": now,
		"location":   location,
	}
	s.emitEvent(ctx, "ShipmentCompleted", deliveryEvent)

	log.Printf("Shipment %s completed, product %s delivered to %s", shipmentID, shipment.ProductID, shipment.To)
	return nil
}

// RecallProduct recalls a product
func (s *SupplyChain) RecallProduct(ctx contractapi.TransactionContextInterface, id string, reason string) error {
	product, err := s.GetProduct(ctx, id)
	if err != nil {
		return err
	}

	clientID, err := ctx.GetClientIdentity().GetID()
	if err != nil {
		return fmt.Errorf("failed to get client identity: %v", err)
	}

	// Only manufacturer can recall
	if clientID != product.Manufacturer {
		return fmt.Errorf("only manufacturer can recall product")
	}

	product.Status = StatusRecalled
	product.Timestamp = time.Now()

	err = s.updateProduct(ctx, product)
	if err != nil {
		return err
	}

	// Emit event
	recallEvent := map[string]interface{}{
		"productId": id,
		"reason":    reason,
		"timestamp": product.Timestamp,
	}
	s.emitEvent(ctx, "ProductRecalled", recallEvent)

	log.Printf("Product %s recalled: %s", id, reason)
	return nil
	}

// GetProduct returns a product by ID
func (s *SupplyChain) GetProduct(ctx contractapi.TransactionContextInterface, id string) (*Product, error) {
	productJSON, err := ctx.GetStub().GetState(id)
	if err != nil {
		return nil, fmt.Errorf("failed to read product: %v", err)
	}
	if productJSON == nil {
		return nil, fmt.Errorf("product %s does not exist", id)
	}

	var product Product
	err = json.Unmarshal(productJSON, &product)
	if err != nil {
		return nil, err
	}

	return &product, nil
}

// ProductExists checks if a product exists
func (s *SupplyChain) ProductExists(ctx contractapi.TransactionContextInterface, id string) (bool, error) {
	productJSON, err := ctx.GetStub().GetState(id)
	if err != nil {
		return false, fmt.Errorf("failed to read product: %v", err)
	}
	return productJSON != nil, nil
}

// GetProductHistory returns the complete history of a product
func (s *SupplyChain) GetProductHistory(ctx contractapi.TransactionContextInterface, id string) ([]ProductHistory, error) {
	resultsIterator, err := ctx.GetStub().GetHistoryForKey(id)
	if err != nil {
		return nil, err
	}
	defer resultsIterator.Close()

	var history []ProductHistory
	for resultsIterator.HasNext() {
		response, err := resultsIterator.Next()
		if err != nil {
			return nil, err
		}

		var product Product
		if len(response.Value) > 0 {
			err = json.Unmarshal(response.Value, &product)
			if err != nil {
				return nil, err
			}
		}

		record := ProductHistory{
			Product:   product,
			TxID:      response.TxId,
			Timestamp: time.Unix(response.Timestamp.Seconds, int64(response.Timestamp.Nanos)),
			IsDelete:  response.IsDelete,
		}
		history = append(history, record)
	}

	return history, nil
}

// GetProductsByManufacturer returns all products by manufacturer
func (s *SupplyChain) GetProductsByManufacturer(ctx contractapi.TransactionContextInterface, manufacturer string) ([]*Product, error) {
	queryString := fmt.Sprintf(`{"selector":{"manufacturer":"%s"}}`, manufacturer)
	return s.queryProducts(ctx, queryString)
}

// GetProductsByOwner returns all products by current owner
func (s *SupplyChain) GetProductsByOwner(ctx contractapi.TransactionContextInterface, owner string) ([]*Product, error) {
	queryString := fmt.Sprintf(`{"selector":{"currentOwner":"%s"}}`, owner)
	return s.queryProducts(ctx, queryString)
}

// GetProductsByStatus returns all products with a specific status
func (s *SupplyChain) GetProductsByStatus(ctx contractapi.TransactionContextInterface, status string) ([]*Product, error) {
	queryString := fmt.Sprintf(`{"selector":{"status":"%s"}}`, status)
	return s.queryProducts(ctx, queryString)
}

// GetAllProducts returns all products
func (s *SupplyChain) GetAllProducts(ctx contractapi.TransactionContextInterface) ([]*Product, error) {
	resultsIterator, err := ctx.GetStub().GetStateByRange("", "")
	if err != nil {
		return nil, err
	}
	defer resultsIterator.Close()

	var products []*Product
	for resultsIterator.HasNext() {
		queryResponse, err := resultsIterator.Next()
		if err != nil {
			return nil, err
		}

		// Skip shipment records
		if len(queryResponse.Key) > 9 && queryResponse.Key[:9] == "shipment_" {
			continue
		}

		var product Product
		err = json.Unmarshal(queryResponse.Value, &product)
		if err != nil {
			// Skip if not a product
			continue
		}
		products = append(products, &product)
	}

	return products, nil
}

// GetShipment returns a shipment by ID
func (s *SupplyChain) GetShipment(ctx contractapi.TransactionContextInterface, id string) (*Shipment, error) {
	shipmentKey := fmt.Sprintf("shipment_%s", id)
	shipmentJSON, err := ctx.GetStub().GetState(shipmentKey)
	if err != nil {
		return nil, fmt.Errorf("failed to read shipment: %v", err)
	}
	if shipmentJSON == nil {
		return nil, fmt.Errorf("shipment %s does not exist", id)
	}

	var shipment Shipment
	err = json.Unmarshal(shipmentJSON, &shipment)
	if err != nil {
		return nil, err
	}

	return &shipment, nil
}

// VerifyProvenance verifies the complete chain of custody
func (s *SupplyChain) VerifyProvenance(ctx contractapi.TransactionContextInterface, id string) (bool, error) {
	history, err := s.GetProductHistory(ctx, id)
	if err != nil {
		return false, err
	}

	if len(history) == 0 {
		return false, fmt.Errorf("no history found for product %s", id)
	}

	// Check that product was manufactured
	firstRecord := history[0]
	if firstRecord.Product.Status != StatusManufactured {
		return false, nil
	}

	// Verify manufacturer hasn't changed
	manufacturer := firstRecord.Product.Manufacturer
	for _, record := range history {
		if record.Product.Manufacturer != manufacturer {
			return false, nil
		}
	}

	return true, nil
}

// Helper functions

func (s *SupplyChain) updateProduct(ctx contractapi.TransactionContextInterface, product *Product) error {
	productJSON, err := json.Marshal(product)
	if err != nil {
		return err
	}
	return ctx.GetStub().PutState(product.ID, productJSON)
}

func (s *SupplyChain) queryProducts(ctx contractapi.TransactionContextInterface, queryString string) ([]*Product, error) {
	resultsIterator, err := ctx.GetStub().GetQueryResult(queryString)
	if err != nil {
		return nil, err
	}
	defer resultsIterator.Close()

	var products []*Product
	for resultsIterator.HasNext() {
		queryResponse, err := resultsIterator.Next()
		if err != nil {
			return nil, err
		}

		var product Product
		err = json.Unmarshal(queryResponse.Value, &product)
		if err != nil {
			return nil, err
		}
		products = append(products, &product)
	}

	return products, nil
}

func (s *SupplyChain) emitProductEvent(ctx contractapi.TransactionContextInterface, eventName string, product Product) {
	productJSON, _ := json.Marshal(product)
	ctx.GetStub().SetEvent(eventName, productJSON)
}

func (s *SupplyChain) emitEvent(ctx contractapi.TransactionContextInterface, eventName string, data interface{}) {
	eventJSON, _ := json.Marshal(data)
	ctx.GetStub().SetEvent(eventName, eventJSON)
}

func main() {
	chaincode, err := contractapi.NewChaincode(&SupplyChain{})
	if err != nil {
		log.Panicf("Error creating supply-chain chaincode: %v", err)
	}

	if err := chaincode.Start(); err != nil {
		log.Panicf("Error starting supply-chain chaincode: %v", err)
	}
}