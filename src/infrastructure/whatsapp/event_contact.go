package whatsapp

import (
	"context"
	"strings"
	"time"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

func handleContactUpdate(ctx context.Context, evt *events.Contact, sessionID, deviceID string, client *whatsmeow.Client) {
	if evt.FromFullSync || evt.Timestamp.IsZero() || evt.Action == nil {
		return
	}
	go func() {
		webhookCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 30*time.Second)
		defer cancel()
		if err := forwardContactUpdate(webhookCtx, evt, sessionID, deviceID, client); err != nil {
			log.Errorf("Failed to forward contact update to webhook: %v", err)
		}
	}()
}

func forwardContactUpdate(ctx context.Context, evt *events.Contact, sessionID, deviceID string, client *whatsmeow.Client) error {
	name := strings.TrimSpace(evt.Action.GetFullName())
	if name == "" {
		name = strings.TrimSpace(evt.Action.GetFirstName())
	}
	jid := evt.JID.ToNonAD()
	if name == "" || (jid.Server != types.DefaultUserServer && jid.Server != types.HiddenUserServer) {
		return nil
	}

	payload := map[string]any{"chat_id": jid.String(), "name": name}
	if phone := NormalizeJIDFromLID(ctx, jid, client).ToNonAD(); phone != jid && phone.Server == types.DefaultUserServer {
		payload["phone_jid"] = phone.String()
	}
	return forwardPayloadToConfiguredWebhooks(ctx, map[string]any{
		"event":      "contact.updated",
		"session_id": sessionID,
		"device_id":  deviceID,
		"timestamp":  evt.Timestamp.Format(time.RFC3339Nano),
		"payload":    payload,
	}, "contact.updated")
}
