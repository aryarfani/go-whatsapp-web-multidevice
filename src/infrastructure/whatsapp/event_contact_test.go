package whatsapp

import (
	"context"
	"testing"
	"time"

	"go.mau.fi/whatsmeow/proto/waSyncAction"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

func TestForwardContactUpdate(t *testing.T) {
	captured := captureGroupWebhooks(t)
	fullName, firstName := "  Saved contact XC  ", "Saved"
	evt := &events.Contact{
		JID:       types.JID{User: "628111", Server: types.DefaultUserServer, Device: 2},
		Timestamp: time.Date(2026, 11, 27, 8, 51, 23, 1000000, time.UTC),
		Action:    &waSyncAction.ContactAction{FullName: &fullName, FirstName: &firstName},
	}
	if err := forwardContactUpdate(context.Background(), evt, "7", "", nil); err != nil {
		t.Fatal(err)
	}
	if len(*captured) != 1 {
		t.Fatalf("expected one webhook, got %d", len(*captured))
	}
	got := (*captured)[0]
	if got.event != "contact.updated" || got.payload["chat_id"] != "628111@s.whatsapp.net" ||
		got.payload["name"] != "Saved contact XC" {
		t.Fatalf("unexpected webhook: %+v", got)
	}
}
