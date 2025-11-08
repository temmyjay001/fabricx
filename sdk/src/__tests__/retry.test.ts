// sdk/src/__tests__/retry.test.ts
import { RetryManager, RetryStrategy } from '../utils/retry';
import { FabricXError } from '../types';

describe('RetryManager', () => {
  describe('execute', () => {
    it('should succeed on first attempt', async () => {
      const retryManager = new RetryManager({ maxAttempts: 3 });
      const operation = jest.fn().mockResolvedValue('success');

      const result = await retryManager.execute(operation);

      expect(result).toBe('success');
      expect(operation).toHaveBeenCalledTimes(1);
    });

    it('should retry on failure and eventually succeed', async () => {
      const retryManager = new RetryManager({
        maxAttempts: 3,
        initialDelay: 10,
      });

      let attempts = 0;
      const operation = jest.fn().mockImplementation(() => {
        attempts++;
        if (attempts < 3) {
          throw new FabricXError('Temporary error', 'CONNECTION_ERROR');
        }
        return Promise.resolve('success');
      });

      const result = await retryManager.execute(operation);

      expect(result).toBe('success');
      expect(operation).toHaveBeenCalledTimes(3);
    });

    it('should fail after max attempts', async () => {
      const retryManager = new RetryManager({
        maxAttempts: 3,
        initialDelay: 10,
      });

      const operation = jest.fn().mockRejectedValue(
        new FabricXError('Permanent error', 'CONNECTION_ERROR')
      );

      await expect(retryManager.execute(operation)).rejects.toThrow('Operation failed after 3 attempts: Permanent error');
      expect(operation).toHaveBeenCalledTimes(3);
    });

    it('should not retry non-retryable errors', async () => {
      const retryManager = new RetryManager({
        maxAttempts: 3,
        initialDelay: 10,
      });

      const operation = jest.fn().mockRejectedValue(
        new FabricXError('Invalid argument', 'INVALID_ARGUMENT')
      );

      await expect(retryManager.execute(operation)).rejects.toThrow('Invalid argument');
      expect(operation).toHaveBeenCalledTimes(1);
    });
  });

  describe('Retry strategies', () => {
    it('should use fixed delay', async () => {
      const retryManager = new RetryManager({
        maxAttempts: 3,
        initialDelay: 100,
        strategy: RetryStrategy.FIXED,
      });

      let attempts = 0;
      const startTime = Date.now();

      const operation = jest.fn().mockImplementation(() => {
        attempts++;
        if (attempts < 3) {
          throw new FabricXError('Error', 'CONNECTION_ERROR');
        }
        return Promise.resolve('success');
      });

      await retryManager.execute(operation);
      const duration = Date.now() - startTime;

      // Should take approximately 200ms (2 retries * 100ms)
      expect(duration).toBeGreaterThanOrEqual(190);
      expect(duration).toBeLessThan(300);
    });

    it('should use exponential backoff', async () => {
      const retryManager = new RetryManager({
        maxAttempts: 3,
        initialDelay: 100,
        strategy: RetryStrategy.EXPONENTIAL,
        backoffMultiplier: 2,
        jitter: false,
      });

      let attempts = 0;
      const startTime = Date.now();

      const operation = jest.fn().mockImplementation(() => {
        attempts++;
        if (attempts < 3) {
          throw new FabricXError('Error', 'CONNECTION_ERROR');
        }
        return Promise.resolve('success');
      });

      await retryManager.execute(operation);
      const duration = Date.now() - startTime;

      // Should take approximately 300ms (100ms + 200ms)
      expect(duration).toBeGreaterThanOrEqual(290);
      expect(duration).toBeLessThan(400);
    });
  });

  describe('Predefined policies', () => {
    it('should use conservative policy', async () => {
      const operation = jest.fn().mockResolvedValue('success');

      const result = await RetryManager.Policies.conservative.execute(operation);

      expect(result).toBe('success');
    });

    it('should use aggressive policy', async () => {
      const operation = jest.fn().mockResolvedValue('success');

      const result = await RetryManager.Policies.aggressive.execute(operation);

      expect(result).toBe('success');
    });
  });
});