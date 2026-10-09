package ocppj

import (
	"errors"
	"strings"
	"testing"
)

func TestParseCall(t *testing.T) {
	m, err := Parse([]byte(`[2,"19223201","BootNotification",{"chargePointVendor":"VendorX","chargePointModel":"SingleSocket"}]`))
	if err != nil {
		t.Fatal(err)
	}
	if m.Type != Call || m.ID != "19223201" || m.Action != "BootNotification" {
		t.Fatalf("unexpected: %+v", m)
	}
	if !strings.Contains(string(m.Payload), "VendorX") {
		t.Fatalf("payload lost: %s", m.Payload)
	}
}

func TestParseCallResult(t *testing.T) {
	m, err := Parse([]byte(`[3,"19223201",{"status":"Accepted","interval":300}]`))
	if err != nil {
		t.Fatal(err)
	}
	if m.Type != CallResult || m.ID != "19223201" {
		t.Fatalf("unexpected: %+v", m)
	}
}

func TestParseCallError(t *testing.T) {
	m, err := Parse([]byte(`[4,"42","NotImplemented","no such action",{}]`))
	if err != nil {
		t.Fatal(err)
	}
	if m.Type != CallError || m.ErrorCode != NotImplemented || m.Description != "no such action" {
		t.Fatalf("unexpected: %+v", m)
	}
}

func TestParseRejectsMalformedFrames(t *testing.T) {
	cases := map[string]string{
		"not json":          `hello`,
		"object not array":  `{"a":1}`,
		"too short":         `[2,"1"]`,
		"unknown type":      `[9,"1",{}]`,
		"call missing body": `[2,"1","Heartbeat"]`,
		"call no action":    `[2,"1","",{}]`,
		"result extra":      `[3,"1",{},{}]`,
		"error short":       `[4,"1","GenericError","x"]`,
		"empty id":          `[3,"",{}]`,
		"id too long":       `[3,"` + strings.Repeat("a", 37) + `",{}]`,
		"id not string":     `[3,5,{}]`,
		"type not number":   `["2","1","X",{}]`,
	}
	for name, frame := range cases {
		if _, err := Parse([]byte(frame)); err == nil {
			t.Errorf("%s: expected an error for %s", name, frame)
		}
	}
}

func TestUnknownTypeIsDistinguishable(t *testing.T) {
	_, err := Parse([]byte(`[9,"1",{}]`))
	if !errors.Is(err, ErrUnknownType) {
		t.Fatalf("want ErrUnknownType, got %v", err)
	}
}

func TestRoundTrip(t *testing.T) {
	frame, err := NewCall("abc", "Heartbeat", nil)
	if err != nil {
		t.Fatal(err)
	}
	if string(frame) != `[2,"abc","Heartbeat",{}]` {
		t.Fatalf("got %s", frame)
	}
	m, err := Parse(frame)
	if err != nil || m.Action != "Heartbeat" {
		t.Fatalf("round trip failed: %+v %v", m, err)
	}

	res, _ := NewCallResult("abc", map[string]string{"status": "Accepted"})
	if string(res) != `[3,"abc",{"status":"Accepted"}]` {
		t.Fatalf("got %s", res)
	}

	ce, _ := NewCallError("abc", GenericError, "boom")
	m, err = Parse(ce)
	if err != nil || m.ErrorCode != GenericError || m.Description != "boom" {
		t.Fatalf("round trip failed: %+v %v", m, err)
	}
}
