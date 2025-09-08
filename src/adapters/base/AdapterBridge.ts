import { TestRunnerAdapter } from './TestRunnerAdapter';

/**
 * Bridge pattern to adapt specific test runner interfaces to our standardized interface
 */
export abstract class AdapterBridge<T> implements TestRunnerAdapter {
  protected nativeAdapter: T;
  
  constructor(nativeAdapter: T) {
    this.nativeAdapter = nativeAdapter;
  }
  
  /** Map native test runner hooks to our standardized interface */
  abstract mapNativeHooks(): void;
  
  /** Extract test status from native test runner result */
  abstract extractTestStatus(result: any): 'PASS' | 'FAIL' | 'SKIP';
  
  // Default implementations that can be overridden
  initialize(): void {
    // Override in specific adapters
  }
  
  startCapture(): void {
    // Override in specific adapters
  }
  
  stopCapture(): void {
    // Override in specific adapters
  }
  
  handleTestFileStart(filePath: string): void {
    // Override in specific adapters
  }
  
  handleTestFileResult(filePath: string, status: 'PASS' | 'FAIL' | 'SKIP'): void {
    // Override in specific adapters
  }
  
  cleanup(): void {
    // Override in specific adapters
  }
}