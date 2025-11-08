// sdk/src/utils/logger.ts

/**
 * Log level enumeration
 */
export enum LogLevel {
  DEBUG = 'debug',
  INFO = 'info',
  WARN = 'warn',
  ERROR = 'error',
  SILENT = 'silent',
}

/**
 * Log entry structure
 */
export interface LogEntry {
  timestamp: Date;
  level: LogLevel;
  message: string;
  context?: any;
}

/**
 * Logger interface
 */
export interface ILogger {
  debug(message: string, context?: any): void;
  info(message: string, context?: any): void;
  warn(message: string, context?: any): void;
  error(message: string, context?: any): void;
  setLevel(level: LogLevel): void;
  getLevel(): LogLevel;
}

/**
 * Production-grade logger for FabricX SDK
 */
export class Logger implements ILogger {
  private level: LogLevel;
  private enableConsole: boolean;
  private logs: LogEntry[] = [];
  private maxLogs: number = 1000;
  private logHandlers: Array<(entry: LogEntry) => void> = [];

  constructor(level: LogLevel = LogLevel.INFO, enableConsole: boolean = true) {
    this.level = level;
    this.enableConsole = enableConsole;
  }

  /**
   * Log a debug message
   */
  debug(message: string, context?: any): void {
    this.log(LogLevel.DEBUG, message, context);
  }

  /**
   * Log an info message
   */
  info(message: string, context?: any): void {
    this.log(LogLevel.INFO, message, context);
  }

  /**
   * Log a warning message
   */
  warn(message: string, context?: any): void {
    this.log(LogLevel.WARN, message, context);
  }

  /**
   * Log an error message
   */
  error(message: string, context?: any): void {
    this.log(LogLevel.ERROR, message, context);
  }

  /**
   * Set log level
   */
  setLevel(level: LogLevel): void {
    this.level = level;
  }

  /**
   * Get current log level
   */
  getLevel(): LogLevel {
    return this.level;
  }

  /**
   * Add a custom log handler
   */
  addHandler(handler: (entry: LogEntry) => void): void {
    this.logHandlers.push(handler);
  }

  /**
   * Remove a log handler
   */
  removeHandler(handler: (entry: LogEntry) => void): void {
    const index = this.logHandlers.indexOf(handler);
    if (index > -1) {
      this.logHandlers.splice(index, 1);
    }
  }

  /**
   * Get all logs
   */
  getLogs(): LogEntry[] {
    return [...this.logs];
  }

  /**
   * Clear all logs
   */
  clearLogs(): void {
    this.logs = [];
  }

  /**
   * Get logs by level
   */
  getLogsByLevel(level: LogLevel): LogEntry[] {
    return this.logs.filter((entry) => entry.level === level);
  }

  /**
   * Get logs in a time range
   */
  getLogsByTimeRange(start: Date, end: Date): LogEntry[] {
    return this.logs.filter((entry) => entry.timestamp >= start && entry.timestamp <= end);
  }

  /**
   * Export logs as JSON
   */
  exportLogs(): string {
    return JSON.stringify(this.logs, null, 2);
  }

  /**
   * Core logging function
   */
  private log(level: LogLevel, message: string, context?: any): void {
    // Check if we should log this level
    if (!this.shouldLog(level)) {
      return;
    }

    const entry: LogEntry = {
      timestamp: new Date(),
      level,
      message,
      context,
    };

    // Store log entry
    this.logs.push(entry);

    // Trim logs if exceeding max
    if (this.logs.length > this.maxLogs) {
      this.logs.shift();
    }

    // Console output
    if (this.enableConsole) {
      this.logToConsole(entry);
    }

    // Call custom handlers
    this.logHandlers.forEach((handler) => {
      try {
        handler(entry);
      } catch (error) {
        // Ignore handler errors to prevent breaking the logger
        console.error('Log handler error:', error);
      }
    });
  }

  /**
   * Check if a log level should be logged
   */
  private shouldLog(level: LogLevel): boolean {
    if (this.level === LogLevel.SILENT) {
      return false;
    }

    const levels = [LogLevel.DEBUG, LogLevel.INFO, LogLevel.WARN, LogLevel.ERROR];
    const currentLevelIndex = levels.indexOf(this.level);
    const logLevelIndex = levels.indexOf(level);

    return logLevelIndex >= currentLevelIndex;
  }

  /**
   * Log to console with formatting
   */
  private logToConsole(entry: LogEntry): void {
    const timestamp = entry.timestamp.toISOString();
    const prefix = `[${timestamp}] [${entry.level.toUpperCase()}]`;
    const message = entry.message;
    const context = entry.context ? `\n${JSON.stringify(entry.context, null, 2)}` : '';

    switch (entry.level) {
      case LogLevel.DEBUG:
        console.debug(`${prefix} ${message}${context}`);
        break;
      case LogLevel.INFO:
        console.info(`${prefix} ${message}${context}`);
        break;
      case LogLevel.WARN:
        console.warn(`${prefix} ${message}${context}`);
        break;
      case LogLevel.ERROR:
        console.error(`${prefix} ${message}${context}`);
        break;
    }
  }

  /**
   * Set maximum number of logs to store
   */
  setMaxLogs(max: number): void {
    this.maxLogs = max;

    // Trim existing logs if needed
    if (this.logs.length > max) {
      this.logs = this.logs.slice(this.logs.length - max);
    }
  }

  /**
   * Enable or disable console output
   */
  setConsoleEnabled(enabled: boolean): void {
    this.enableConsole = enabled;
  }

  /**
   * Create a child logger with a prefix
   */
  createChild(prefix: string): Logger {
    const childLogger = new Logger(this.level, this.enableConsole);

    // Add handler to prefix messages
    childLogger.addHandler((entry) => {
      this.log(entry.level, `[${prefix}] ${entry.message}`, entry.context);
    });

    return childLogger;
  }
}

/**
 * Create a logger that writes to a file (Node.js only)
 */
export class FileLogger extends Logger {
  private filePath?: string;
  private writeStream?: any;

  constructor(level: LogLevel, enableConsole: boolean, filePath?: string) {
    super(level, enableConsole);

    if (filePath && typeof require !== 'undefined') {
      this.filePath = filePath;
      this.setupFileLogging();
    }
  }

  private setupFileLogging(): void {
    try {
      const fs = require('fs');
      const path = require('path');

      if (this.filePath) {
        // Ensure directory exists
        const dir = path.dirname(this.filePath);
        if (!fs.existsSync(dir)) {
          fs.mkdirSync(dir, { recursive: true });
        }

        // Create write stream
        this.writeStream = fs.createWriteStream(this.filePath, { flags: 'a' });

        // Add handler to write to file
        this.addHandler((entry) => {
          if (this.writeStream) {
            const line = JSON.stringify(entry) + '\n';
            this.writeStream.write(line);
          }
        });
      }
    } catch (error) {
      console.error('Failed to setup file logging:', error);
    }
  }

  /**
   * Close the file stream
   */
  async close(): Promise<void> {
    if (this.writeStream) {
      return new Promise((resolve) => {
        this.writeStream.end(() => {
          resolve();
        });
      });
    }
  }
}

/**
 * Create a logger that sends logs to a remote server
 */
export class RemoteLogger extends Logger {
  private endpoint?: string;
  private batchSize: number = 10;
  private flushInterval: number = 5000;
  private buffer: LogEntry[] = [];
  private flushTimer?: NodeJS.Timeout;

  constructor(
    level: LogLevel,
    enableConsole: boolean,
    endpoint?: string,
    options?: { batchSize?: number; flushInterval?: number }
  ) {
    super(level, enableConsole);

    if (endpoint) {
      this.endpoint = endpoint;
      this.batchSize = options?.batchSize || 10;
      this.flushInterval = options?.flushInterval || 5000;

      this.setupRemoteLogging();
    }
  }

  private setupRemoteLogging(): void {
    // Add handler to buffer logs
    this.addHandler((entry) => {
      this.buffer.push(entry);

      if (this.buffer.length >= this.batchSize) {
        this.flush();
      }
    });

    // Setup periodic flush
    this.flushTimer = setInterval(() => {
      if (this.buffer.length > 0) {
        this.flush();
      }
    }, this.flushInterval);
  }

  /**
   * Flush buffered logs to remote server
   */
  private async flush(): Promise<void> {
    if (!this.endpoint || this.buffer.length === 0) {
      return;
    }

    const logsToSend = [...this.buffer];
    this.buffer = [];

    try {
      await fetch(this.endpoint, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({ logs: logsToSend }),
      });
    } catch (error) {
      console.error('Failed to send logs to remote server:', error);
      // Re-add logs to buffer for retry
      this.buffer.unshift(...logsToSend);
    }
  }

  /**
   * Stop remote logging and flush remaining logs
   */
  async close(): Promise<void> {
    if (this.flushTimer) {
      clearInterval(this.flushTimer);
    }

    await this.flush();
  }
}
