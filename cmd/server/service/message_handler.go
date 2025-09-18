// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package service

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/linuxfoundation/lfx-v2-auth-service/internal/domain/port"
	"github.com/linuxfoundation/lfx-v2-auth-service/pkg/constants"
	"github.com/linuxfoundation/lfx-v2-auth-service/pkg/log"
)

// MessageHandlerService handles NATS messages using the service layer
type MessageHandlerService struct {
	messageHandler port.MessageHandler
}

// HandleMessage routes NATS messages to appropriate handlers
func (mhs *MessageHandlerService) HandleMessage(ctx context.Context, msg port.TransportMessenger) {
	subject := msg.Subject()
	ctx = log.AppendCtx(ctx, slog.String("subject", subject))

	slog.DebugContext(ctx, "handling NATS message")

	handlers := map[string]func(ctx context.Context, msg port.TransportMessenger) ([]byte, error){
		constants.UserUpdateSubject: mhs.messageHandler.UpdateUser,
	}

	handler, ok := handlers[subject]
	if !ok {
		slog.WarnContext(ctx, "unknown subject")
		mhs.respondWithError(ctx, msg, "unknown subject")
		return
	}

	response, errHandler := handler(ctx, msg)
	if errHandler != nil {
		slog.ErrorContext(ctx, "error handling message",
			"error", errHandler,
			"subject", subject,
		)
		mhs.respondWithError(ctx, msg, errHandler.Error())
		return
	}

	errRespond := msg.Respond(response)
	if errRespond != nil {
		slog.ErrorContext(ctx, "error responding to NATS message", "error", errRespond)
		return
	}

	slog.DebugContext(ctx, "responded to NATS message", "response", string(response))
}

func (mhs *MessageHandlerService) respondWithError(ctx context.Context, msg port.TransportMessenger, errorMsg string) {
	errResponse := []byte(fmt.Sprintf(`{"error":"%s"}`, errorMsg))
	if err := msg.Respond(errResponse); err != nil {
		slog.ErrorContext(ctx, "failed to send error response", "error", err)
	}
}

// NewMessageHandlerService creates a new message handler service
func NewMessageHandlerService(messageHandler port.MessageHandler) *MessageHandlerService {
	return &MessageHandlerService{
		messageHandler: messageHandler,
	}
}
