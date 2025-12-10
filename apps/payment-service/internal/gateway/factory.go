package gateway

import (
	"fmt"
	"strings"
)

// GatewayType represents the type of payment gateway
type GatewayType string

const (
	GatewayTypeMock   GatewayType = "mock"
	GatewayTypeStripe GatewayType = "stripe"
)

// NewPaymentGateway creates a payment gateway based on the type
func NewPaymentGateway(gatewayType string, config *GatewayConfig) (PaymentGateway, error) {
	switch GatewayType(strings.ToLower(gatewayType)) {
	case GatewayTypeMock, "":
		// Default to mock gateway
		return NewMockGateway(DefaultMockGatewayConfig()), nil

	case GatewayTypeStripe:
		if config == nil || config.SecretKey == "" {
			return nil, fmt.Errorf("stripe secret key is required")
		}
		return NewStripeGateway(&StripeGatewayConfig{
			SecretKey:     config.SecretKey,
			WebhookSecret: config.WebhookSecret,
			Environment:   config.Environment,
		})

	default:
		return nil, fmt.Errorf("unsupported gateway type: %s", gatewayType)
	}
}

// NewMockGatewayWithConfig creates a mock gateway with custom configuration
func NewMockGatewayWithConfig(successRate float64, delayMs int) PaymentGateway {
	return NewMockGateway(&MockGatewayConfig{
		SuccessRate: successRate,
		DelayMs:     delayMs,
		FailureReasons: []string{
			"insufficient_funds",
			"card_declined",
			"expired_card",
			"processing_error",
		},
	})
}
