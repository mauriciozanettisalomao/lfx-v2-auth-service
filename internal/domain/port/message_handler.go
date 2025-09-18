// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package port

import "context"

// MessageHandler defines the behavior of the message handler
type MessageHandler interface {
	UpdateUser(ctx context.Context, msg TransportMessenger) ([]byte, error)
}
