package comfyUIclient

import (
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
)

type WebSocketConnection struct {
	URL         string
	Conn        *websocket.Conn
	isConnected atomic.Bool
	MaxRetry    int
	handler     Handler
	BearerToken string
}

type Handler interface {
	Handle(string) error
}

func NewDefaultWebSocketConnection(url string, handler Handler, bearerToken string) *WebSocketConnection {
	return NewWebSocketConnection(url, 3, handler, bearerToken)
}

func NewWebSocketConnection(url string, maxRetry int, handler Handler, bearerToken string) *WebSocketConnection {
	return &WebSocketConnection{
		URL:         url,
		MaxRetry:    maxRetry,
		handler:     handler,
		BearerToken: bearerToken,
	}
}

// ConnectAndListen connects to the websocket and listens for messages
func (w *WebSocketConnection) ConnectAndListen() {
	defer w.Close()
	for {
		if !w.GetIsConnected() {
			var err error
			for i := 0; i < w.MaxRetry; i++ {
				if err = w.Connect(); err != nil {
					fmt.Printf("[%s] websocket connection error %v\n", w.URL, err)
					continue
				}
				break
			}

			if err == nil {
				w.SetIsConnected(true)
				go w.listen()
			}
		}
		time.Sleep(5 * time.Second)
	}
}

func (w *WebSocketConnection) Connect() error {
	var err error
	var headers map[string][]string

	if w.BearerToken != "" {
		headers = map[string][]string{
			"Authorization": {"Bearer " + w.BearerToken},
		}
	}

	w.Conn, _, err = websocket.DefaultDialer.Dial(w.URL, headers)
	if err != nil {
		return fmt.Errorf("[%s] websocket.DefaultDialer.Dial: error: %w", w.URL, err)
	}
	w.SetIsConnected(true)
	return nil
}

func (w *WebSocketConnection) listen() {
	defer w.Close()
	for {
		_, message, err := w.Conn.ReadMessage()
		if err != nil {
			w.SetIsConnected(false)
			break
		}

		w.handler.Handle(string(message))
	}

}

func (w *WebSocketConnection) Close() error {
	if err := w.Conn.Close(); err != nil {
		return fmt.Errorf(" w.Conn.Close() error: %w", err)
	}
	return nil
}

func (w *WebSocketConnection) GetIsConnected() bool {
	return w.isConnected.Load()
}

func (w *WebSocketConnection) SetIsConnected(iConnected bool) {
	w.isConnected.Store(iConnected)
}

type WSMessage struct {
	Type WsMessageType `json:"type"`
	Data interface{}   `json:"data"`
}

var (
	messageTypeMap map[WsMessageType]func() interface{}
	once           sync.Once
)

func getWSMessageData(messageType WsMessageType) interface{} {
	once.Do(func() {
		messageTypeMap = map[WsMessageType]func() interface{}{
			Status:               func() interface{} { return &WSMessageDataStatus{} },
			ExecutionStart:       func() interface{} { return &WSMessageDataExecutionStart{} },
			ExecutionCached:      func() interface{} { return &WSMessageDataExecutionCached{} },
			Executing:            func() interface{} { return &WSMessageDataExecuting{} },
			Progress:             func() interface{} { return &WSMessageDataProgress{} },
			Executed:             func() interface{} { return &WSMessageDataExecuted{} },
			ExecutionInterrupted: func() interface{} { return &WSMessageExecutionInterrupted{} },
			ExecutionError:       func() interface{} { return &WSMessageExecutionError{} },
			ExecutionSuccess:     func() interface{} { return &WSMessageExecuteSuccess{} },
		}
	})

	fn, exist := messageTypeMap[messageType]
	if !exist {
		return &WSEmptyMessage{}
	}
	return fn()
}

func (m *WSMessage) UnmarshalJSON(b []byte) error {
	var temp struct {
		Type WsMessageType   `json:"type"`
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(b, &temp); err != nil {
		return err
	}

	m.Type = temp.Type
	messageData := getWSMessageData(m.Type)
	if messageData != nil {
		if err := json.Unmarshal(temp.Data, messageData); err != nil {
			return err
		}
		m.Data = messageData
	}
	return nil
}

//	WSMessageDataStatus
//
// Json {"type": "status", "data": {"status": {"exec_info": {"queue_remaining": 1}}}}
type WSMessageDataStatus struct {
	Status struct {
		ExecInfo struct {
			QueueRemaining int `json:"queue_remaining"`
		} `json:"exec_info"`
	} `json:"status"`
	SID string `json:"sid"`
}

// WSMessageDataExecutionStart
// Json {"type": "execution_start", "data": {"prompt_id": "ed986d60-2a27-4d28-8871-2fdb36582902"}}
type WSMessageDataExecutionStart struct {
	PromptID string `json:"prompt_id"`
}

// WSMessageDataExecutionCached
// json {"type": "execution_cached", "data": {"nodes": [], "prompt_id": "ed986d60-2a27-4d28-8871-2fdb36582902"}}
type WSMessageDataExecutionCached struct {
	Nodes    []string `json:"nodes"`
	PromptID string   `json:"prompt_id"`
}

// WSMessageDataExecuting
// json {"type": "executing", "data": {"node": "12", "prompt_id": "ed986d60-2a27-4d28-8871-2fdb36582902"}}
type WSMessageDataExecuting struct {
	Node     string `json:"node"`
	PromptID string `json:"prompt_id"`
}

// WSMessageDataProgress
/*
{
  "type": "progress",
  "data": {
    "value": 18,
    "max": 20
  }
}
*/
type WSMessageDataProgress struct {
	Value int `json:"value"`
	Max   int `json:"max"`
}

//
/*
{"type": "executed", "data": {"node": "19", "output": {"images": [{"filename": "ComfyUI_00046_.png", "subfolder": "", "type": "output"}]}, "prompt_id": "ed986d60-2a27-4d28-8871-2fdb36582902"}}

// when there are multiple outputs, each output will receive an "executed"
{"type": "executed", "data": {"node": "53", "output": {"images": [{"filename": "ComfyUI_temp_mynbi_00001_.png", "subfolder": "", "type": "temp"}]}, "prompt_id": "3bcf5bac-19e1-4219-a0eb-50a84e4db2ea"}}
{"type": "executed", "data": {"node": "19", "output": {"images": [{"filename": "ComfyUI_00052_.png", "subfolder": "", "type": "output"}]}, "prompt_id": "3bcf5bac-19e1-4219-a0eb-50a84e4db2ea"}}
*/

type WSMessageDataExecuted struct {
	Node     string `json:"node"`
	PromptID string `json:"prompt_id"`
	Output   map[string][]*DataOutputFile
}

// WSMessageExecutionInterrupted
/*
{"type": "execution_interrupted", "data": {"prompt_id": "dc7093d7-980a-4fe6-bf0c-f6fef932c74b", "node_id": "19", "node_type": "SaveImage", "executed": ["5", "17", "10", "11"]}}
*/
type WSMessageExecutionInterrupted struct {
	PromptID string   `json:"prompt_id"`
	NodeID   string   `json:"node_id"`
	NodeType string   `json:"node_type"`
	Executed []string `json:"executed"`
}

type WSMessageExecuteSuccess struct {
	PromptID string `json:"prompt_id"`
}

type WSEmptyMessage struct {
}

type WSMessageExecutionError struct {
	PromptID         string                 `json:"prompt_id"`
	Node             string                 `json:"node_id"`
	NodeType         string                 `json:"node_type"`
	Executed         []string               `json:"executed"`
	ExceptionMessage string                 `json:"exception_message"`
	ExceptionType    string                 `json:"exception_type"`
	Traceback        []string               `json:"traceback"`
	CurrentInputs    map[string]interface{} `json:"current_inputs"`
	CurrentOutputs   map[int]interface{}    `json:"current_outputs"`
}
