# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

ComfyUIClient is a Go client library for interacting with the ComfyUI API. It provides a comprehensive wrapper around ComfyUI's REST and WebSocket APIs for managing image generation workflows.

## Development Commands

### Build and Test
```bash
# Get dependencies
go get -u github.com/XdpCs/comfyUIclient

# Run tests (if any exist)
go test ./...

# Build the library
go build

# Run examples
cd examples/textToImage && go run main.go
cd examples/api && go run main.go
```

### Module Information
- Package name: `github.com/kee-moo/comfyUIclient`
- Go version: 1.18+
- Main dependencies:
  - `github.com/google/uuid` - Client ID generation
  - `github.com/gorilla/websocket` - WebSocket connections

## Architecture

### Core Components

#### Client (`client.go`)
The `Client` struct is the main entry point and manages both HTTP and WebSocket connections:
- **Initialization**: Use `NewDefaultClient(endPoint)` or `NewDefaultClientStr(baseURL)` to create clients
- **WebSocket**: Automatically manages WebSocket connection for real-time task updates
- **Client ID**: Each client has a unique UUID that ties it to its WebSocket session
- **Authorization**: Set tokens via `SetEASToken(token)` for authenticated requests

#### WebSocket Connection (`websocket.go`)
- Maintains persistent connection with auto-reconnect (default: 3 retries)
- Handles message routing through the `Handler` interface
- Client implements `Handler.Handle(msg string)` to process incoming WebSocket messages
- Messages are sent to the client's task status channel for consumption

#### Entity Types (`entity.go`)
Key data structures:
- `PromptNode`: Workflow node definition with inputs and class type
- `PromptHistoryItem`: Execution history with outputs (images, gifs, audio, video)
- `QueuePromptResp`: Response from queueing a prompt, includes prompt_id
- `SystemStats`: Server system and GPU information
- `DataOutputFile`: File reference with filename, subfolder, and type

#### Message Types (`const.go`)
WebSocket message types tracked:
- `Status`: Queue status updates
- `ExecutionStart`: Workflow execution begins
- `Executing`: Current node being executed
- `Progress`: Progress within a node (value/max)
- `Executed`: Node completed with outputs
- `ExecutionError`: Execution failed
- `ExecutionSuccess`: Workflow completed successfully

### Client Workflow Pattern

1. **Initialize and Connect**:
```go
client := comfyUIclient.NewDefaultClient(endPoint)
client.ConnectAndListen()  // Start WebSocket in goroutine
for !client.IsInitialized() {}  // Wait for connection
```

2. **Queue Prompts**:
   - `QueuePromptByString(workflow, extraData)`: JSON string workflow
   - `QueuePromptByNodes(nodes, extraData)`: Map of PromptNode objects

3. **Monitor Execution**:
```go
for taskStatus := range client.GetTaskStatus() {
    switch taskStatus.Type {
    case comfyUIclient.Executed:
        // Get output files
    }
}
```

### API Mapping

The client wraps these ComfyUI REST endpoints:
- **Queue Management**: `/queue` - Delete queues, get queue info
- **Prompt Execution**: `/prompt` - Queue prompts, get remaining count
- **History**: `/history` - Get/delete execution history
- **Files**: `/view` - Download generated images/files
- **System**: `/system_stats`, `/object_info` - Server info and node schemas
- **Uploads**: `/upload/image`, `/upload/mask` - Upload input images
- **Control**: `/interrupt` - Stop execution

### Important Implementation Details

1. **Client ID Binding**: Queue operations (delete) only work on prompts queued by the same client ID
2. **HTTP Client**: Default timeout is 10 seconds, customizable via `NewClient(endPoint, httpClient)`
3. **Authorization Header**: Token is set as `Authorization` header when provided
4. **Protocol Detection**: Automatically converts `https://` base URLs to `wss://` for WebSocket
5. **Message Unmarshaling**: Custom JSON unmarshaling for `NodeInfo` (array format) and `WSMessage` (type-based polymorphism)

## File Organization

- `client.go` - Main client and HTTP request methods
- `websocket.go` - WebSocket connection management and message types
- `entity.go` - Data structures for API requests/responses
- `const.go` - Constants for routers, message types, and image types
- `examples/` - Usage examples (textToImage, api)
