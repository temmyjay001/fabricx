import { Context, Contract, Info, Returns, Transaction } from 'fabric-contract-api';

interface TokenMetadata {
  name: string;
  symbol: string;
  decimals: number;
  totalSupply: number;
  owner: string;
}

@Info({ title: 'ERC20Token', description: 'ERC-20 token implementation' })
export class ERC20TokenContract extends Contract {
  @Transaction()
  public async Initialize(
    ctx: Context,
    name: string,
    symbol: string,
    decimals: number,
    initialSupply: number
  ): Promise<void> {
    // Check if already initialized
    const exists = await this.TokenExists(ctx);
    if (exists) {
      throw new Error('Token already initialized');
    }

    const clientId = ctx.clientIdentity.getID();

    const metadata: TokenMetadata = {
      name,
      symbol,
      decimals,
      totalSupply: initialSupply,
      owner: clientId,
    };

    await ctx.stub.putState('metadata', Buffer.from(JSON.stringify(metadata)));

    // Mint initial supply to owner
    await this.mint(ctx, clientId, initialSupply);
  }

  @Transaction(false)
  public async TokenExists(ctx: Context): Promise<boolean> {
    const metadataBytes = await ctx.stub.getState('metadata');
    return metadataBytes && metadataBytes.length > 0;
  }

  @Transaction(false)
  @Returns('string')
  public async Name(ctx: Context): Promise<string> {
    const metadata = await this.getMetadata(ctx);
    return metadata.name;
  }

  @Transaction(false)
  @Returns('string')
  public async Symbol(ctx: Context): Promise<string> {
    const metadata = await this.getMetadata(ctx);
    return metadata.symbol;
  }

  @Transaction(false)
  @Returns('number')
  public async Decimals(ctx: Context): Promise<number> {
    const metadata = await this.getMetadata(ctx);
    return metadata.decimals;
  }

  @Transaction(false)
  @Returns('number')
  public async TotalSupply(ctx: Context): Promise<number> {
    const metadata = await this.getMetadata(ctx);
    return metadata.totalSupply;
  }

  @Transaction(false)
  @Returns('number')
  public async BalanceOf(ctx: Context, account: string): Promise<number> {
    const balanceKey = `balance_${account}`;
    const balanceBytes = await ctx.stub.getState(balanceKey);

    if (!balanceBytes || balanceBytes.length === 0) {
      return 0;
    }

    return parseInt(balanceBytes.toString());
  }

  @Transaction()
  public async Transfer(ctx: Context, to: string, amount: number): Promise<void> {
    const from = ctx.clientIdentity.getID();
    await this.transferHelper(ctx, from, to, amount);
  }

  @Transaction()
  public async TransferFrom(ctx: Context, from: string, to: string, amount: number): Promise<void> {
    const spender = ctx.clientIdentity.getID();

    // Check allowance
    const allowance = await this.Allowance(ctx, from, spender);
    if (allowance < amount) {
      throw new Error(`Insufficient allowance: have ${allowance}, need ${amount}`);
    }

    // Transfer tokens
    await this.transferHelper(ctx, from, to, amount);

    // Update allowance
    const newAllowance = allowance - amount;
    await this.setAllowance(ctx, from, spender, newAllowance);
  }

  @Transaction()
  public async Approve(ctx: Context, spender: string, amount: number): Promise<void> {
    const owner = ctx.clientIdentity.getID();
    await this.setAllowance(ctx, owner, spender, amount);

    // Emit approval event
    const event = {
      owner,
      spender,
      amount,
    };
    ctx.stub.setEvent('Approval', Buffer.from(JSON.stringify(event)));
  }

  @Transaction(false)
  @Returns('number')
  public async Allowance(ctx: Context, owner: string, spender: string): Promise<number> {
    const allowanceKey = `allowance_${owner}_${spender}`;
    const allowanceBytes = await ctx.stub.getState(allowanceKey);

    if (!allowanceBytes || allowanceBytes.length === 0) {
      return 0;
    }

    return parseInt(allowanceBytes.toString());
  }

  @Transaction()
  public async Mint(ctx: Context, to: string, amount: number): Promise<void> {
    const clientId = ctx.clientIdentity.getID();
    const metadata = await this.getMetadata(ctx);

    if (clientId !== metadata.owner) {
      throw new Error('Only owner can mint tokens');
    }

    await this.mint(ctx, to, amount);

    // Update total supply
    metadata.totalSupply += amount;
    await ctx.stub.putState('metadata', Buffer.from(JSON.stringify(metadata)));
  }

  @Transaction()
  public async Burn(ctx: Context, amount: number): Promise<void> {
    const account = ctx.clientIdentity.getID();
    const balance = await this.BalanceOf(ctx, account);

    if (balance < amount) {
      throw new Error(`Insufficient balance: have ${balance}, need ${amount}`);
    }

    // Reduce balance
    const newBalance = balance - amount;
    await this.setBalance(ctx, account, newBalance);

    // Update total supply
    const metadata = await this.getMetadata(ctx);
    metadata.totalSupply -= amount;
    await ctx.stub.putState('metadata', Buffer.from(JSON.stringify(metadata)));

    // Emit transfer event to zero address
    const event = {
      from: account,
      to: '0x0',
      amount,
    };
    ctx.stub.setEvent('Transfer', Buffer.from(JSON.stringify(event)));
  }

  // Helper methods
  private async getMetadata(ctx: Context): Promise<TokenMetadata> {
    const metadataBytes = await ctx.stub.getState('metadata');
    if (!metadataBytes || metadataBytes.length === 0) {
      throw new Error('Token not initialized');
    }
    return JSON.parse(metadataBytes.toString());
  }

  private async setBalance(ctx: Context, account: string, balance: number): Promise<void> {
    const balanceKey = `balance_${account}`;
    await ctx.stub.putState(balanceKey, Buffer.from(balance.toString()));
  }

  private async setAllowance(
    ctx: Context,
    owner: string,
    spender: string,
    amount: number
  ): Promise<void> {
    const allowanceKey = `allowance_${owner}_${spender}`;
    await ctx.stub.putState(allowanceKey, Buffer.from(amount.toString()));
  }

  private async mint(ctx: Context, to: string, amount: number): Promise<void> {
    const balance = await this.BalanceOf(ctx, to);
    const newBalance = balance + amount;
    await this.setBalance(ctx, to, newBalance);

    // Emit transfer event from zero address
    const event = {
      from: '0x0',
      to,
      amount,
    };
    ctx.stub.setEvent('Transfer', Buffer.from(JSON.stringify(event)));
  }

  private async transferHelper(
    ctx: Context,
    from: string,
    to: string,
    amount: number
  ): Promise<void> {
    if (amount <= 0) {
      throw new Error('Transfer amount must be positive');
    }

    // Get sender balance
    const fromBalance = await this.BalanceOf(ctx, from);
    if (fromBalance < amount) {
      throw new Error(`Insufficient balance: have ${fromBalance}, need ${amount}`);
    }

    // Get recipient balance
    const toBalance = await this.BalanceOf(ctx, to);

    // Update balances
    await this.setBalance(ctx, from, fromBalance - amount);
    await this.setBalance(ctx, to, toBalance + amount);

    // Emit transfer event
    const event = {
      from,
      to,
      amount,
    };
    ctx.stub.setEvent('Transfer', Buffer.from(JSON.stringify(event)));
  }
}
