package whatsapp

import (
	"context"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waSyncAction"
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

// callLogRecordOf returns the call log record carried by a "call_log" app-state mutation.
func callLogRecordOf(evt *events.AppState) *waSyncAction.CallLogRecord {
	if len(evt.Index) == 0 || evt.Index[0] != "call_log" {
		return nil
	}
	return evt.GetCallLogAction().GetCallLogRecord()
}

// handleCallLog forwards call.log: the phone's final call record (result, duration, video),
// which unlike the live call.* events also covers how a connected call ended.
func handleCallLog(ctx context.Context, rec *waSyncAction.CallLogRecord, deviceID string, client *whatsmeow.Client) {
	go func() {
		webhookCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 30*time.Second)
		defer cancel()
		payload := createCallLogPayload(webhookCtx, rec, deviceID, client)
		if err := forwardPayloadToConfiguredWebhooks(webhookCtx, payload, "call.log"); err != nil {
			logrus.Errorf("Failed to forward call.log to webhook: %v", err)
		}
	}()
}

func createCallLogPayload(ctx context.Context, rec *waSyncAction.CallLogRecord, deviceID string, client *whatsmeow.Client) map[string]any {
	payload := map[string]any{
		"call_id":     rec.GetCallID(),
		"call_result": strings.ToLower(rec.GetCallResult().String()),
		"duration":    rec.GetDuration(),
		"is_incoming": rec.GetIsIncoming(),
		"is_video":    rec.GetIsVideo(),
		"call_type":   strings.ToLower(rec.GetCallType().String()),
	}
	if creator, err := types.ParseJID(rec.GetCallCreatorJID()); err == nil && !creator.IsEmpty() {
		if creator.Server == "lid" {
			payload["from_lid"] = creator.ToNonAD().String()
		}
		payload["from"] = NormalizeJIDFromLID(ctx, creator, client).ToNonAD().String()
	}
	if group, err := types.ParseJID(rec.GetGroupJID()); err == nil && !group.IsEmpty() {
		payload["group_jid"] = group.ToNonAD().String()
	}

	startedAt := time.Now()
	if rec.StartTime != nil {
		startedAt = time.Unix(rec.GetStartTime(), 0)
	}
	payload["start_time"] = startedAt.UTC().Format(time.RFC3339)

	body := map[string]any{"event": "call.log", "timestamp": startedAt.UTC().Format(time.RFC3339), "payload": payload}
	if deviceID != "" {
		body["device_id"] = deviceID
	}
	return body
}
