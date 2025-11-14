# Asset Transfer Chaincode

Basic CRUD operations for asset management with ownership tracking.

## Features

- ✅ Create, Read, Update, Delete assets
- ✅ Transfer asset ownership
- ✅ Query assets by owner
- ✅ Get asset history (all changes)
- ✅ Event emission on all operations
- ✅ Timestamps for creation and updates

## Data Model

```go
type Asset struct {
    ID             string    // Unique identifier
    Owner          string    // Current owner
    Value          int       // Asset value
    Color          string    // Asset color
    Size           int       // Asset size
    AppraisedValue int       // Appraised value
    CreatedAt      time.Time // Creation timestamp
    UpdatedAt      time.Time // Last update timestamp
}
```

## Functions

### InitLedger

Initialize the ledger with sample assets.

```bash
npx fabricx invoke asset-transfer InitLedger
```

### CreateAsset

Create a new asset.

```bash
npx fabricx invoke asset-transfer CreateAsset asset6 Frank 800 purple 20 800
```

### ReadAsset

Read an asset by ID.

```bash
npx fabricx query asset-transfer ReadAsset asset1
```

### UpdateAsset

Update an existing asset.

```bash
npx fabricx invoke asset-transfer UpdateAsset asset1 Alice 350 blue 6 350
```

### DeleteAsset

Delete an asset.

```bash
npx fabricx invoke asset-transfer DeleteAsset asset1
```

### TransferAsset

Transfer asset ownership.

```bash
npx fabricx invoke asset-transfer TransferAsset asset1 Bob
```

### GetAllAssets

Get all assets.

```bash
npx fabricx query asset-transfer GetAllAssets
```

### GetAssetsByOwner

Get all assets owned by a specific owner.

```bash
npx fabricx query asset-transfer GetAssetsByOwner Alice
```

### GetAssetHistory

Get the complete history of an asset.

```bash
npx fabricx query asset-transfer GetAssetHistory asset1
```

## Events

The chaincode emits the following events:

- `AssetCreated` - When a new asset is created
- `AssetUpdated` - When an asset is updated
- `AssetDeleted` - When an asset is deleted
- `AssetTransferred` - When asset ownership changes

## Testing

```bash
# Run unit tests
go test -v

# Run with coverage
go test -v -cover
```

## Use Cases

- Supply chain tracking
- Inventory management
- Document management
- Equipment tracking
- Real estate records
