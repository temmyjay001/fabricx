# Supply Chain Chaincode

Track products through supply chain with provenance verification.

## Features

- ✅ Product creation and tracking
- ✅ Ownership transfer
- ✅ Status management
- ✅ Shipment tracking
- ✅ Complete history and provenance
- ✅ Product recall capability
- ✅ Multi-party verification

## Product Lifecycle

```
MANUFACTURED → IN_TRANSIT → RECEIVED → INSPECTED → DELIVERED
                                          ↓
                                      RECALLED
```

## Data Model

```go
type Product struct {
    ID           string        // Unique identifier
    Name         string        // Product name
    Description  string        // Description
    Manufacturer string        // Original manufacturer
    CurrentOwner string        // Current owner
    Status       ProductStatus // Current status
    Location     string        // Current location
    Timestamp    time.Time     // Last update time
    Metadata     string        // Additional metadata
}

type Shipment struct {
    ID           string    // Shipment ID
    ProductID    string    // Product being shipped
    From         string    // Sender
    To           string    // Recipient
    Carrier      string    // Shipping carrier
    StartedAt    time.Time // Shipment start
    ExpectedAt   time.Time // Expected delivery
    DeliveredAt  *time.Time // Actual delivery
    TrackingInfo string    // Tracking number
}
```

## Functions

### CreateProduct

Create a new product.

```bash
npx fabricx invoke supply-chain CreateProduct product1 "Laptop" "Dell XPS 15" "Factory A" "{\"serial\":\"ABC123\"}"
```

### TransferProduct

Transfer product ownership.

```bash
npx fabricx invoke supply-chain TransferProduct product1 distributor1 "Warehouse B"
```

### UpdateStatus

Update product status.

```bash
npx fabricx invoke supply-chain UpdateStatus product1 INSPECTED "Quality Control Dept"
```

### CreateShipment

Create a shipment.

```bash
npx fabricx invoke supply-chain CreateShipment shipment1 product1 retailer1 "FedEx" 3 "TRACK123456"
```

### CompleteShipment

Mark shipment as delivered.

```bash
npx fabricx invoke supply-chain CompleteShipment shipment1 "Store Location"
```

### RecallProduct

Recall a product (manufacturer only).

```bash
npx fabricx invoke supply-chain RecallProduct product1 "Safety issue detected"
```

### Query Functions

```bash
# Get specific product
npx fabricx query supply-chain GetProduct product1

# Get product history
npx fabricx query supply-chain GetProductHistory product1

# Get by manufacturer
npx fabricx query supply-chain GetProductsByManufacturer manufacturer1

# Get by owner
npx fabricx query supply-chain GetProductsByOwner owner1

# Get by status
npx fabricx query supply-chain GetProductsByStatus IN_TRANSIT

# Get all products
npx fabricx query supply-chain GetAllProducts

# Get shipment
npx fabricx query supply-chain GetShipment shipment1

# Verify provenance
npx fabricx query supply-chain VerifyProvenance product1
```

## Events

- `ProductCreated` - New product created
- `ProductTransferred` - Ownership transferred
- `StatusUpdated` - Status changed
- `ShipmentCreated` - Shipment initiated
- `ShipmentCompleted` - Shipment delivered
- `ProductRecalled` - Product recalled

## Use Cases

- Product tracking
- Authenticity verification
- Compliance monitoring
- Recall management
- Supply chain visibility
- Quality control
