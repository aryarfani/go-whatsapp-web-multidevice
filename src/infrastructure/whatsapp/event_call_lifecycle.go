package whatsapp

import (
	"context"
	"time"

	"github.com/sirupsen/logrus"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

// handleCallAccept forwards call.accept. For an incoming call this fires when any
// device of this account (including a phone) answers, so event_from identifies it.
func handleCallAccept(ctx context.Context, evt *events.CallAccept, deviceID string, client *whatsmeow.Client) {
	forwardCallLifecycle(ctx, "call.accept", evt.BasicCallMeta, "", deviceID, client)
}

// handleCallTerminate forwards call.terminate, which carries the end reason.
func handleCallTerminate(ctx context.Context, evt *events.CallTerminate, deviceID string, client *whatsmeow.Client) {
	forwardCallLifecycle(ctx, "call.terminate", evt.BasicCallMeta, evt.Reason, deviceID, client)
}

// handleCallReject forwards call.reject.
func handleCallReject(ctx context.Context, evt *events.CallReject, deviceID string, client *whatsmeow.Client) {
	forwardCallLifecycle(ctx, "call.reject", evt.BasicCallMeta, "", deviceID, client)
}

func forwardCallLifecycle(ctx context.Context, eventName string, meta types.BasicCallMeta, reason, deviceID string, client *whatsmeow.Client) {
	logrus.Infof("%s (CallID: %s, reason: %q)", eventName, meta.CallID, reason)
	go func() {
		webhookCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 30*time.Second)
		defer cancel()
		payload := createCallLifecyclePayload(webhookCtx, eventName, meta, reason, deviceID, client)
		if err := forwardPayloadToConfiguredWebhooks(webhookCtx, payload, eventName); err != nil {
			logrus.Errorf("Failed to forward %s to webhook: %v", eventName, err)
		}
	}()
}

// createCallLifecyclePayload mirrors the call.offer payload: "from" is the call creator
// (same as call.offer, so events join on call_id), "event_from" is the JID that sent this
// particular event.
func createCallLifecyclePayload(ctx context.Context, eventName string, meta types.BasicCallMeta, reason, deviceID string, client *whatsmeow.Client) map[string]any {
	payload := map[string]any{"call_id": meta.CallID}

	creator := meta.CallCreator
	if creator.Server == "lid" {
		payload["from_lid"] = creator.ToNonAD().String()
	}
	payload["from"] = NormalizeJIDFromLID(ctx, creator, client).ToNonAD().String()
	if !meta.From.IsEmpty() {
		payload["event_from"] = meta.From.ToNonAD().String()
	}
	if !meta.GroupJID.IsEmpty() {
		payload["group_jid"] = meta.GroupJID.ToNonAD().String()
	}
	if reason != "" {
		payload["reason"] = reason
	}

	body := map[string]any{
		"event":     eventName,
		"timestamp": meta.Timestamp.Format(time.RFC3339),
		"payload":   payload,
	}
	if deviceID != "" {
		body["device_id"] = deviceID
	}
	return body
}
