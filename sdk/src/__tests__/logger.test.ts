// sdk/src/__tests__/logger.test.ts
import { Logger, LogLevel } from '../utils/logger';

describe('Logger', () => {
  let logger: Logger;

  beforeEach(() => {
    logger = new Logger(LogLevel.INFO, false);
  });

  it('should log messages at appropriate levels', () => {
    logger.debug('Debug message');
    logger.info('Info message');
    logger.warn('Warning message');
    logger.error('Error message');

    const logs = logger.getLogs();
    expect(logs).toHaveLength(3); // Debug should be filtered out
    expect(logs[0].level).toBe(LogLevel.INFO);
    expect(logs[1].level).toBe(LogLevel.WARN);
    expect(logs[2].level).toBe(LogLevel.ERROR);
  });

  it('should filter logs by level', () => {
    logger.setLevel(LogLevel.WARN);

    logger.debug('Debug');
    logger.info('Info');
    logger.warn('Warning');
    logger.error('Error');

    const logs = logger.getLogs();
    expect(logs).toHaveLength(2);
  });

  it('should get logs by specific level', () => {
    logger.info('Info 1');
    logger.warn('Warning 1');
    logger.info('Info 2');

    const infoLogs = logger.getLogsByLevel(LogLevel.INFO);
    expect(infoLogs).toHaveLength(2);
  });

  it('should clear logs', () => {
    logger.info('Message');
    expect(logger.getLogs()).toHaveLength(1);

    logger.clearLogs();
    expect(logger.getLogs()).toHaveLength(0);
  });

  it('should add custom handlers', () => {
    const customHandler = jest.fn();
    logger.addHandler(customHandler);

    logger.info('Test message');

    expect(customHandler).toHaveBeenCalled();
  });
});