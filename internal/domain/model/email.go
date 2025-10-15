// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package model

// Email represents an email
type Email struct {
	OTP           string `json:"otp"`
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
}
