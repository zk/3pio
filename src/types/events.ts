export interface StdoutChunkEvent {
  eventType: 'stdoutChunk';
  payload: {
    filePath: string;
    chunk: string;
  };
}

export interface StderrChunkEvent {
  eventType: 'stderrChunk';
  payload: {
    filePath: string;
    chunk: string;
  };
}

export interface TestFileStartEvent {
  eventType: 'testFileStart';
  payload: {
    filePath: string;
  };
}

export interface TestFileResultEvent {
  eventType: 'testFileResult';
  payload: {
    filePath: string;
    status: 'PASS' | 'FAIL' | 'SKIP';
    failedTests?: Array<{
      name: string;
      duration?: number;
    }>;
  };
}

export type IPCEvent = StdoutChunkEvent | StderrChunkEvent | TestFileStartEvent | TestFileResultEvent;

export interface TestRunState {
  timestamp: string;
  status: 'RUNNING' | 'COMPLETE' | 'ERROR';
  updatedAt: string;
  arguments: string;
  totalFiles: number;
  filesCompleted: number;
  filesPassed: number;
  filesFailed: number;
  filesSkipped: number;
  testFiles: Array<{
    status: 'PENDING' | 'RUNNING' | 'PASS' | 'FAIL' | 'SKIP';
    file: string;
    logFile?: string;
  }>;
}