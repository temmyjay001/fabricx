// sdk/src/utils/retry.ts
import { FabricXError } from '../types';

/**
 * Retry strategy enumeration
 */
export enum RetryStrategy {
  /** Fixed delay between retries */
  FIXED = 'fixed',
  /** Exponential backoff with optional jitter */
  EXPONENTIAL = 'exponential',
  /** Linear backoff */
  LINEAR = 'linear',
}

/**
 * Retry options configuration
 */
export interface RetryOptions {
  /** Maximum number of retry attempts (default: 3) */
  maxAttempts?: number;
  /** Initial delay in milliseconds (default: 1000) */
  initialDelay?: number;
  /** Maximum delay in milliseconds (default: 30000) */
  maxDelay?: number;
  /** Retry strategy (default: EXPONENTIAL) */
  strategy?: RetryStrategy;
  /** Backoff multiplier for exponential/linear strategies (default: 2) */
  backoffMultiplier?: number;
  /** Add random jitter to delay (default: true) */
  jitter?: boolean;
  /** Jitter factor (0-1, default: 0.1) */
  jitterFactor?: number;
  /** List of error codes that should trigger a retry */
  retryableErrors?: string[];
  /** Callback when retry attempt is made */
  onRetry?: (attempt: number, error: Error) => void;
}

/**
 * Retry result metadata
 */
export interface RetryMetadata {
  attempts: number;
  totalDelay: number;
  lastError?: Error;
}

/**
 * Production-grade retry manager with multiple strategies
 */
export class RetryManager {
  private config: Required<RetryOptions>;

  constructor(options?: RetryOptions) {
    this.config = {
      maxAttempts: options?.maxAttempts || 3,
      initialDelay: options?.initialDelay || 1000,
      maxDelay: options?.maxDelay || 30000,
      strategy: options?.strategy || RetryStrategy.EXPONENTIAL,
      backoffMultiplier: options?.backoffMultiplier || 2,
      jitter: options?.jitter !== false,
      jitterFactor: options?.jitterFactor || 0.1,
      retryableErrors: options?.retryableErrors || [
        'CONNECTION_ERROR',
        'TIMEOUT',
        'UNAVAILABLE',
        'TRANSIENT_FAILURE',
        'RESOURCE_EXHAUSTED',
      ],
      onRetry: options?.onRetry || (() => {}),
    };
  }

  /**
   * Execute an operation with retry logic
   */
  async execute<T>(operation: () => Promise<T>, options?: Partial<RetryOptions>): Promise<T> {
    const config = { ...this.config, ...options };
    let attempt = 0;
    let totalDelay = 0;
    let lastError: Error | undefined;

    while (attempt < config.maxAttempts) {
      try {
        return await operation();
      } catch (error) {
        attempt++;
        lastError = error as Error;

        // Check if error is retryable
        if (!this.isRetryable(error as Error, config.retryableErrors)) {
          throw error;
        }

        // If this was the last attempt, throw the error
        if (attempt >= config.maxAttempts) {
          throw new FabricXError(
            `Operation failed after ${attempt} attempts: ${lastError.message}`,
            'MAX_RETRIES_EXCEEDED',
            {
              attempts: attempt,
              totalDelay,
              lastError,
            }
          );
        }

        // Calculate delay
        const delay = this.calculateDelay(attempt, config);
        totalDelay += delay;

        // Call onRetry callback
        config.onRetry(attempt, lastError);

        // Wait before retry
        await this.sleep(delay);
      }
    }

    // This should never be reached, but TypeScript needs it
    throw new FabricXError('Unexpected retry error', 'RETRY_ERROR', {
      attempts: attempt,
      totalDelay,
      lastError,
    });
  }

  /**
   * Execute with custom retry predicate
   */
  async executeWithPredicate<T>(
    operation: () => Promise<T>,
    shouldRetry: (error: Error, attempt: number) => boolean,
    options?: Partial<RetryOptions>
  ): Promise<T> {
    const config = { ...this.config, ...options };
    let attempt = 0;
    let totalDelay = 0;
    let lastError: Error | undefined;

    while (attempt < config.maxAttempts) {
      try {
        return await operation();
      } catch (error) {
        attempt++;
        lastError = error as Error;

        // Check custom predicate
        if (!shouldRetry(lastError, attempt)) {
          throw error;
        }

        // If this was the last attempt, throw the error
        if (attempt >= config.maxAttempts) {
          throw new FabricXError(
            `Operation failed after ${attempt} attempts: ${lastError.message}`,
            'MAX_RETRIES_EXCEEDED',
            {
              attempts: attempt,
              totalDelay,
              lastError,
            }
          );
        }

        // Calculate delay
        const delay = this.calculateDelay(attempt, config);
        totalDelay += delay;

        // Call onRetry callback
        config.onRetry(attempt, lastError);

        // Wait before retry
        await this.sleep(delay);
      }
    }

    throw new FabricXError('Unexpected retry error', 'RETRY_ERROR', {
      attempts: attempt,
      totalDelay,
      lastError,
    });
  }

  /**
   * Check if an error is retryable
   */
  private isRetryable(error: Error, retryableErrors: string[]): boolean {
    if (error instanceof FabricXError) {
      return retryableErrors.includes(error.code);
    }

    // Check error message for retryable patterns
    const message = error.message.toLowerCase();
    const retryablePatterns = [
      'timeout',
      'unavailable',
      'connection',
      'network',
      'econnrefused',
      'enotfound',
      'etimedout',
    ];

    return retryablePatterns.some((pattern) => message.includes(pattern));
  }

  /**
   * Calculate delay based on retry strategy
   */
  private calculateDelay(attempt: number, config: Required<RetryOptions>): number {
    let delay: number;

    switch (config.strategy) {
      case RetryStrategy.FIXED:
        delay = config.initialDelay;
        break;

      case RetryStrategy.LINEAR:
        delay = config.initialDelay * attempt * config.backoffMultiplier;
        break;

      case RetryStrategy.EXPONENTIAL:
        delay = config.initialDelay * Math.pow(config.backoffMultiplier, attempt - 1);
        break;

      default:
        delay = config.initialDelay;
    }

    // Apply max delay cap
    delay = Math.min(delay, config.maxDelay);

    // Apply jitter if enabled
    if (config.jitter) {
      delay = this.addJitter(delay, config.jitterFactor);
    }

    return Math.floor(delay);
  }

  /**
   * Add random jitter to delay
   */
  private addJitter(delay: number, jitterFactor: number): number {
    const jitter = delay * jitterFactor;
    const randomJitter = (Math.random() * 2 - 1) * jitter;
    return delay + randomJitter;
  }

  /**
   * Sleep for specified milliseconds
   */
  private sleep(ms: number): Promise<void> {
    return new Promise((resolve) => setTimeout(resolve, ms));
  }

  /**
   * Create a retry policy for specific operations
   */
  static createPolicy(options: RetryOptions): RetryManager {
    return new RetryManager(options);
  }

  /**
   * Predefined retry policies
   */
  static readonly Policies = {
    /**
     * Conservative policy - fewer retries, longer delays
     */
    conservative: new RetryManager({
      maxAttempts: 2,
      initialDelay: 2000,
      maxDelay: 10000,
      strategy: RetryStrategy.EXPONENTIAL,
      backoffMultiplier: 2,
    }),

    /**
     * Aggressive policy - more retries, shorter delays
     */
    aggressive: new RetryManager({
      maxAttempts: 5,
      initialDelay: 500,
      maxDelay: 5000,
      strategy: RetryStrategy.EXPONENTIAL,
      backoffMultiplier: 1.5,
    }),

    /**
     * Network policy - optimized for network errors
     */
    network: new RetryManager({
      maxAttempts: 3,
      initialDelay: 1000,
      maxDelay: 30000,
      strategy: RetryStrategy.EXPONENTIAL,
      backoffMultiplier: 2,
      jitter: true,
      retryableErrors: ['CONNECTION_ERROR', 'TIMEOUT', 'UNAVAILABLE', 'NETWORK_ERROR'],
    }),

    /**
     * Quick policy - fast retries for transient errors
     */
    quick: new RetryManager({
      maxAttempts: 3,
      initialDelay: 100,
      maxDelay: 1000,
      strategy: RetryStrategy.LINEAR,
      backoffMultiplier: 1,
      jitter: false,
    }),
  };
}

/**
 * Circuit breaker pattern implementation
 */
export class CircuitBreaker {
  private failures: number = 0;
  private lastFailureTime?: Date;
  private state: 'CLOSED' | 'OPEN' | 'HALF_OPEN' = 'CLOSED';
  private readonly failureThreshold: number;
  private readonly resetTimeout: number;

  constructor(options?: { failureThreshold?: number; resetTimeout?: number }) {
    this.failureThreshold = options?.failureThreshold || 5;
    this.resetTimeout = options?.resetTimeout || 60000; // 1 minute
  }

  /**
   * Execute operation with circuit breaker
   */
  async execute<T>(operation: () => Promise<T>): Promise<T> {
    // Check if circuit should transition to HALF_OPEN
    if (this.state === 'OPEN' && this.shouldAttemptReset()) {
      this.state = 'HALF_OPEN';
    }

    // If circuit is OPEN, reject immediately
    if (this.state === 'OPEN') {
      throw new FabricXError('Circuit breaker is OPEN', 'CIRCUIT_BREAKER_OPEN', {
        failures: this.failures,
        lastFailureTime: this.lastFailureTime,
      });
    }

    try {
      const result = await operation();

      // Operation succeeded, reset if in HALF_OPEN
      if (this.state === 'HALF_OPEN') {
        this.reset();
      }

      return result;
    } catch (error) {
      this.recordFailure();

      // If threshold reached, open the circuit
      if (this.failures >= this.failureThreshold) {
        this.state = 'OPEN';
      }

      throw error;
    }
  }

  /**
   * Record a failure
   */
  private recordFailure(): void {
    this.failures++;
    this.lastFailureTime = new Date();
  }

  /**
   * Reset the circuit breaker
   */
  private reset(): void {
    this.failures = 0;
    this.lastFailureTime = undefined;
    this.state = 'CLOSED';
  }

  /**
   * Check if enough time has passed to attempt reset
   */
  private shouldAttemptReset(): boolean {
    if (!this.lastFailureTime) {
      return false;
    }

    const now = new Date();
    const timeSinceLastFailure = now.getTime() - this.lastFailureTime.getTime();
    return timeSinceLastFailure >= this.resetTimeout;
  }

  /**
   * Get current circuit breaker state
   */
  getState(): 'CLOSED' | 'OPEN' | 'HALF_OPEN' {
    return this.state;
  }

  /**
   * Get failure count
   */
  getFailureCount(): number {
    return this.failures;
  }

  /**
   * Manually reset the circuit breaker
   */
  manualReset(): void {
    this.reset();
  }

  /**
   * Manually open the circuit breaker
   */
  manualOpen(): void {
    this.state = 'OPEN';
  }
}
