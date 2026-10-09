// Package ocppj implements the OCPP-J (JSON over WebSocket) message framing used
// by OCPP 1.6: a JSON array whose first element is the message type.
//
//	CALL        [2, "<uniqueId>", "<action>", {payload}]
//	CALLRESULT  [3, "<uniqueId>", {payload}]
//	CALLERROR   [4, "<uniqueId>", "<errorCode>", "<description>", {details}]
package ocppj

import (
	"encoding/json"
	"errors"
	"fmt"
)

// Subprotocol is the WebSocket subprotocol a 1.6J charge point must offer.
const Subprotocol = "ocpp1.6"

type MessageType int

const (
	Call       MessageType = 2
	CallResult MessageType = 3
	CallError  MessageType = 4
)

// OCPP 1.6 error codes (section 4.3).
const (
	NotImplemented                = "NotImplemented"
	NotSupported                  = "NotSupported"
	InternalError                 = "InternalError"
	ProtocolError                 = "ProtocolError"
	SecurityError                 = "SecurityError"
	FormationViolation            = "FormationViolation"
	PropertyConstraintViolation   = "PropertyConstraintViolation"
	OccurrenceConstraintViolation = "OccurrenceConstraintViolation"
	TypeConstraintViolation       = "TypeConstraintViolation"
	GenericError                  = "GenericError"
)

// Message is a decoded frame. Which fields are set depends on Type.
type Message struct {
	Type        MessageType
	ID          string
	Action      string          // Call only
	Payload     json.RawMessage // Call and CallResult
	ErrorCode   string          // CallError only
	Description string          // CallError only
	Details     json.RawMessage // CallError only
}

var (
	ErrNotArray    = errors.New("ocppj: frame is not a JSON array")
	ErrTooShort    = errors.New("ocppj: frame is too short")
	ErrUnknownType = errors.New("ocppj: unknown message type")
)

// Parse decodes one WebSocket text frame.
func Parse(data []byte) (*Message, error) {
	var parts []json.RawMessage
	if err := json.Unmarshal(data, &parts); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrNotArray, err)
	}
	if len(parts) < 3 {
		return nil, ErrTooShort
	}

	var typ int
	if err := json.Unmarshal(parts[0], &typ); err != nil {
		return nil, fmt.Errorf("ocppj: bad message type: %w", err)
	}
	m := &Message{Type: MessageType(typ)}
	if err := json.Unmarshal(parts[1], &m.ID); err != nil {
		return nil, fmt.Errorf("ocppj: bad unique id: %w", err)
	}
	if m.ID == "" || len(m.ID) > 36 { // spec: max 36 characters
		return nil, errors.New("ocppj: unique id must be 1-36 characters")
	}

	switch m.Type {
	case Call:
		if len(parts) != 4 {
			return nil, fmt.Errorf("ocppj: CALL needs 4 elements, got %d", len(parts))
		}
		if err := json.Unmarshal(parts[2], &m.Action); err != nil || m.Action == "" {
			return nil, errors.New("ocppj: CALL needs an action")
		}
		m.Payload = parts[3]
	case CallResult:
		if len(parts) != 3 {
			return nil, fmt.Errorf("ocppj: CALLRESULT needs 3 elements, got %d", len(parts))
		}
		m.Payload = parts[2]
	case CallError:
		if len(parts) != 5 {
			return nil, fmt.Errorf("ocppj: CALLERROR needs 5 elements, got %d", len(parts))
		}
		if err := json.Unmarshal(parts[2], &m.ErrorCode); err != nil {
			return nil, fmt.Errorf("ocppj: bad error code: %w", err)
		}
		if err := json.Unmarshal(parts[3], &m.Description); err != nil {
			return nil, fmt.Errorf("ocppj: bad error description: %w", err)
		}
		m.Details = parts[4]
	default:
		return nil, fmt.Errorf("%w: %d", ErrUnknownType, typ)
	}
	return m, nil
}

// NewCall builds a CALL frame. payload is marshalled to JSON.
func NewCall(id, action string, payload any) ([]byte, error) {
	return marshalFrame(Call, id, action, payload)
}

// NewCallResult builds a CALLRESULT frame.
func NewCallResult(id string, payload any) ([]byte, error) {
	return marshalFrame(CallResult, id, nil, payload)
}

// NewCallError builds a CALLERROR frame.
func NewCallError(id, code, description string) ([]byte, error) {
	return json.Marshal([]any{CallError, id, code, description, struct{}{}})
}

func marshalFrame(t MessageType, id string, action any, payload any) ([]byte, error) {
	if payload == nil {
		payload = struct{}{}
	}
	if action == nil {
		return json.Marshal([]any{t, id, payload})
	}
	return json.Marshal([]any{t, id, action, payload})
}
