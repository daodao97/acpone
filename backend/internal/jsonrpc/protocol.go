package jsonrpc

import (
	"encoding/json"
	"fmt"
)

const Version = "2.0"

// Request represents a JSON-RPC request
type Request struct {
	JSONRPC string `json:"jsonrpc"`
	ID      int    `json:"id"`
	Method  string `json:"method"`
	Params  any    `json:"params,omitempty"`
}

// Response represents a JSON-RPC response
type Response struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int             `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *Error          `json:"error,omitempty"`
}

// Notification represents a JSON-RPC notification (no ID)
type Notification struct {
	JSONRPC string `json:"jsonrpc"`
	Method  string `json:"method"`
	Params  any    `json:"params,omitempty"`
}

// Error represents a JSON-RPC error
type Error struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

func (e *Error) Error() string {
	return fmt.Sprintf("jsonrpc error %d: %s", e.Code, e.Message)
}

// Standard error codes
const (
	ParseError     = -32700
	InvalidRequest = -32600
	MethodNotFound = -32601
	InvalidParams  = -32602
	InternalError  = -32603
)

// NewRequest creates a new JSON-RPC request
func NewRequest(id int, method string, params any) *Request {
	return &Request{
		JSONRPC: Version,
		ID:      id,
		Method:  method,
		Params:  params,
	}
}

// NewNotification creates a JSON-RPC notification
func NewNotification(method string, params any) *Notification {
	return &Notification{
		JSONRPC: Version,
		Method:  method,
		Params:  params,
	}
}

// NewResponse creates a success response
func NewResponse(id int, result any) *Response {
	var raw json.RawMessage
	if result == nil {
		raw = json.RawMessage("null")
	} else {
		raw, _ = json.Marshal(result)
	}
	return &Response{
		JSONRPC: Version,
		ID:      id,
		Result:  raw,
	}
}

// NewErrorResponse creates an error response
func NewErrorResponse(id int, code int, message string) *Response {
	return &Response{
		JSONRPC: Version,
		ID:      id,
		Error:   &Error{Code: code, Message: message},
	}
}

// Message is a union type for incoming messages
type Message struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      *int            `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *Error          `json:"error,omitempty"`
}

// IsRequest returns true if message is a request
func (m *Message) IsRequest() bool {
	return m.ID != nil && m.Method != ""
}

// IsResponse returns true if message is a response
func (m *Message) IsResponse() bool {
	return m.ID != nil && m.Method == ""
}

// IsNotification returns true if message is a notification
func (m *Message) IsNotification() bool {
	return m.ID == nil && m.Method != ""
}

// ParseParams unmarshals params into target
func (m *Message) ParseParams(target any) error {
	if m.Params == nil {
		return nil
	}
	return json.Unmarshal(m.Params, target)
}

// ParseResult unmarshals result into target
func (m *Message) ParseResult(target any) error {
	if m.Result == nil {
		return nil
	}
	return json.Unmarshal(m.Result, target)
}
