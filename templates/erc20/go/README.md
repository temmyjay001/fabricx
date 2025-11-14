# ERC-20 Token Chaincode

Fungible token standard with minting, transfers, and allowances.

## Features

- ✅ Standard ERC-20 interface
- ✅ Mint and burn functionality
- ✅ Transfer and transferFrom
- ✅ Approve and allowance mechanism
- ✅ Event emission for all operations
- ✅ Owner-based minting control

## Data Model

```go
type TokenMetadata struct {
    Name        string // Token name
    Symbol      string // Token symbol
    Decimals    uint8  // Decimal places
    TotalSupply uint64 // Total token supply
    Owner       string // Contract owner
}
```

## Functions

### Initialize

Initialize a new token.

```bash
npx fabricx invoke erc20 Initialize MyToken MTK 18 1000000
```

### Name, Symbol, Decimals, TotalSupply

Get token metadata.

```bash
npx fabricx query erc20 Name
npx fabricx query erc20 Symbol
npx fabricx query erc20 Decimals
npx fabricx query erc20 TotalSupply
```

### BalanceOf

Get balance of an account.

```bash
npx fabricx query erc20 BalanceOf user1
```

### Transfer

Transfer tokens to another account.

```bash
npx fabricx invoke erc20 Transfer user2 100
```

### Approve

Approve spender to use tokens.

```bash
npx fabricx invoke erc20 Approve user2 500
```

### Allowance

Check remaining allowance.

```bash
npx fabricx query erc20 Allowance user1 user2
```

### TransferFrom

Transfer tokens using allowance.

```bash
npx fabricx invoke erc20 TransferFrom user1 user3 50
```

### Mint

Mint new tokens (owner only).

```bash
npx fabricx invoke erc20 Mint user1 1000
```

### Burn

Burn tokens from your balance.

```bash
npx fabricx invoke erc20 Burn 500
```

## Events

- `Transfer` - Token transfer
- `Approval` - Allowance approval

## Use Cases

- Loyalty points
- Digital currencies
- Voting tokens
- Reward systems
- Stablecoins
