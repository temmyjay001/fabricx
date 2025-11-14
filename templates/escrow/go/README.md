# Escrow Chaincode

Multi-party escrow with time locks and arbitration.

## Features

- ✅ Three-party escrow (buyer, seller, arbiter)
- ✅ Funding and release workflow
- ✅ Dispute resolution mechanism
- ✅ Time-based deadlines
- ✅ Refund capability
- ✅ Event emission for all state changes

## Escrow Lifecycle

```
PENDING → FUNDED → RELEASED
    ↓         ↓
CANCELLED  DISPUTED → RELEASED/REFUNDED
              ↓
          REFUNDED
```

## Functions

### CreateEscrow

Create a new escrow agreement.

```bash
npx fabricx invoke escrow CreateEscrow escrow1 seller1 arbiter1 1000 "Payment for services" 30
```

### FundEscrow

Fund the escrow (buyer only).

```bash
npx fabricx invoke escrow FundEscrow escrow1
```

### ReleaseEscrow

Release funds to seller (buyer only).

```bash
npx fabricx invoke escrow ReleaseEscrow escrow1
```

### RefundEscrow

Refund to buyer (seller or arbiter).

```bash
npx fabricx invoke escrow RefundEscrow escrow1
```

### DisputeEscrow

Raise a dispute (buyer or seller).

```bash
npx fabricx invoke escrow DisputeEscrow escrow1
```

### ResolveDispute

Resolve dispute (arbiter only).

```bash
# Release to seller
npx fabricx invoke escrow ResolveDispute escrow1 true

# Refund to buyer
npx fabricx invoke escrow ResolveDispute escrow1 false
```

### CancelEscrow

Cancel unfunded escrow (buyer only).

```bash
npx fabricx invoke escrow CancelEscrow escrow1
```

### Query Functions

```bash
# Get specific escrow
npx fabricx query escrow GetEscrow escrow1

# Get by buyer
npx fabricx query escrow GetEscrowsByBuyer buyer1

# Get by seller
npx fabricx query escrow GetEscrowsBySeller seller1

# Get by status
npx fabricx query escrow GetEscrowsByStatus FUNDED

# Get all
npx fabricx query escrow GetAllEscrows
```

## Use Cases

- Payment escrow
- Service agreements
- Marketplace transactions
- Freelance payments
- Real estate transactions
