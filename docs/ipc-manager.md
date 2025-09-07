# **Component Design: IPC Manager**

* **Version:** 1.0  
* **Owner:** Core Team  
* **Status:** Final

## **1\. Core Purpose**

The IPC (Inter-Process Communication) Manager provides a simple, reliable, and performant file-based event bus. It allows the Test Runner Adapters (running in a child process) to send a stream of structured events to the CLI Orchestrator (running in the main process).

## **2\. Public API**

The manager exposes two main functions, one for writing and one for reading.

* **createWriter(ipcFilePath: string): (event: IPCEvent) \=\> Promise\<void\>**  
  * Returns an asynchronous function that takes an event object.  
  * When called, this function serializes the event to a JSON string, appends a newline character, and appends the result to the ipcFilePath.  
  * It uses fs.appendFile to ensure atomic writes.  
* **createReader(ipcFilePath: string, onEvent: (event: IPCEvent) \=\> void): { close: () \=\> void }**  
  * Initializes and returns a reader object.  
  * Uses a robust file watcher library (e.g., chokidar) to monitor ipcFilePath for changes.  
  * It maintains an internal state of the last-read byte offset. When the file grows, it reads only the new data, splits it by newlines, parses each line as JSON, and invokes the onEvent callback for each valid event.  
  * Returns a close() method to stop watching the file.

## **3\. Event Schema**

The manager is responsible for handling the following event structures. All communication through the IPC channel must adhere to this schema.

* **stdoutChunk**: For streaming stdout output.  
  {  
    "eventType": "stdoutChunk",  
    "payload": { "filePath": "...", "chunk": "..." }  
  }

* **stderrChunk**: For streaming stderr output.  
  {  
    "eventType": "stderrChunk",  
    "payload": { "filePath": "...", "chunk": "..." }  
  }

* **testFileResult**: For the final result of a single test file.  
  {  
    "eventType": "testFileResult",  
    "payload": { "filePath": "...", "status": "PASS" | "FAIL" }  
  }

## **4\. Failure Modes**

* **File Permissions:** The writer cannot append to the IPC file, or the reader cannot read it.  
* **Malformed JSON:** A line is written to the IPC file that is not valid JSON, which could cause the reader to crash.  
* **IPC File Deleted Mid-Run:** The IPC file is unexpectedly deleted by an external process while the reader is watching it.  
* **Partial Writes:** A very large event write is interrupted, resulting in a partial, unparseable line at the end of the file.

## **5\. Testing Strategy**

* **Unit Tests:**  
  * Test the createWriter function by calling it with a sample event and asserting that the correct JSON string is written to a temporary file.  
  * Test the createReader with a pre-populated event file and assert that it reads all events correctly on initialization.  
  * Test the live-reading logic by starting a watcher on a temp file and then appending new lines to it, asserting that the onEvent callback is invoked with the correct data for only the new events.  
  * Test the error handling by writing malformed JSON to the file and ensuring the reader logs an error but does not crash.