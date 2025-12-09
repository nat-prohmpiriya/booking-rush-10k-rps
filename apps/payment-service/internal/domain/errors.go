package domain

import "errors"

// Common domain errors
var (
	ErrPaymentNotFound      = errors.New("payment not found")
	ErrPaymentAlreadyExists = errors.New("payment already exists for this booking")
	ErrInvalidPaymentStatus = errors.New("invalid payment status")
	ErrInvalidAmount        = errors.New("invalid payment amount")
	ErrPaymentProcessing    = errors.New("payment is currently being processed")
	ErrPaymentFailed        = errors.New("payment processing failed")
	ErrRefundFailed         = errors.New("refund processing failed")
	ErrInvalidPaymentMethod = errors.New("invalid payment method")
	ErrDuplicateTransaction = errors.New("duplicate transaction")
)
