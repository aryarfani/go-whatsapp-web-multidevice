package whatsapp

import (
	"context"
	"testing"
	"time"

	"go.mau.fi/whatsmeow/types"
)

func TestCreateCallLifecyclePayload(t *testing.T) {
	creator := types.NewJID("628111", types.DefaultUserServer)
	other := types.NewADJID("628222", 0, 7)
	at := time.Date(2026, time.November, 24, 9, 0, 0, 0, time.UTC)
	meta := types.BasicCallMeta{From: other, CallCreator: creator, CallID: "call-1", Timestamp: at}

	body := createCallLifecyclePayload(context.Background(), "call.terminate", meta, "timeout", "dev-1", nil)
	payload := body["payload"].(map[string]any)

	if body["event"] != "call.terminate" || body["device_id"] != "dev-1" || body["timestamp"] != at.Format(time.RFC3339) {
		t.Fatalf("unexpected envelope: %#v", body)
	}
	if payload["call_id"] != "call-1" || payload["from"] != creator.String() || payload["reason"] != "timeout" {
		t.Fatalf("unexpected payload: %#v", payload)
	}
	if payload["event_from"] != other.ToNonAD().String() {
		t.Fatalf("event_from should be the non-AD sender, got %v", payload["event_from"])
	}
	if _, ok := payload["group_jid"]; ok {
		t.Fatalf("group_jid must be omitted for 1:1 calls")
	}

	noReason := createCallLifecyclePayload(context.Background(), "call.accept", meta, "", "", nil)
	if _, ok := noReason["payload"].(map[string]any)["reason"]; ok {
		t.Fatalf("empty reason must be omitted")
	}
	if _, ok := noReason["device_id"]; ok {
		t.Fatalf("empty device_id must be omitted")
	}
}
