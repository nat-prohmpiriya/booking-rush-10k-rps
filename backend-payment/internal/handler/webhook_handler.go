package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prohmpiriya/booking-rush-10k-rps/backend-payment/internal/dto"
	"github.com/prohmpiriya/booking-rush-10k-rps/backend-payment/internal/service"
	"github.com/prohmpiriya/booking-rush-10k-rps/pkg/kafka"
	"github.com/prohmpiriya/booking-rush-10k-rps/pkg/logger"
	"github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/webhook"
)

// WebhookHandler handles Stripe webhook events
type WebhookHandler struct {
	paymentService service.PaymentService
	webhookSecret  string
	kafkaProducer  *kafka.Producer
}

// NewWebhookHandler creates a new WebhookHandler
func NewWebhookHandler(paymentService service.PaymentService, webhookSecret string, kafkaProducer *kafka.Producer) *WebhookHandler {
	return &WebhookHandler{
		paymentService: paymentService,
		webhookSecret:  webhookSecret,
		kafkaProducer:  kafkaProducer,
	}
}

// HandleStripeWebhook handles incoming Stripe webhook events
func (h *WebhookHandler) HandleStripeWebhook(c *gin.Context) {
	log := logger.Get()

	// Read request body
	payload, err := io.ReadAll(c.Request.Body)
	if err != nil {
		log.Error(fmt.Sprintf("Failed to read webhook body: %v", err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to read request body"})
		return
	}

	// Get Stripe signature header
	sigHeader := c.GetHeader("Stripe-Signature")
	if sigHeader == "" {
		log.Warn("Missing Stripe-Signature header")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing Stripe-Signature header"})
		return
	}

	// Verify webhook signature
	event, err := webhook.ConstructEvent(payload, sigHeader, h.webhookSecret)
	if err != nil {
		log.Error(fmt.Sprintf("Failed to verify webhook signature: %v", err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid signature"})
		return
	}

	log.Info(fmt.Sprintf("Received Stripe webhook event: %s", event.Type))

	// Handle different event types
	switch event.Type {
	case "payment_intent.succeeded":
		h.handlePaymentIntentSucceeded(c, event)
	case "payment_intent.payment_failed":
		h.handlePaymentIntentFailed(c, event)
	case "payment_intent.canceled":
		h.handlePaymentIntentCanceled(c, event)
	case "charge.refunded":
		h.handleChargeRefunded(c, event)
	default:
		log.Info(fmt.Sprintf("Unhandled event type: %s", event.Type))
		c.JSON(http.StatusOK, gin.H{"received": true, "message": "Event type not handled"})
		return
	}
}

// handlePaymentIntentSucceeded handles successful payment
func (h *WebhookHandler) handlePaymentIntentSucceeded(c *gin.Context, event stripe.Event) {
	log := logger.Get()

	var paymentIntent stripe.PaymentIntent
	if err := json.Unmarshal(event.Data.Raw, &paymentIntent); err != nil {
		log.Error(fmt.Sprintf("Failed to parse payment_intent.succeeded: %v", err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse event data"})
		return
	}

	paymentID := paymentIntent.Metadata["payment_id"]
	bookingID := paymentIntent.Metadata["booking_id"]

	log.Info(fmt.Sprintf("Payment succeeded: payment_id=%s, booking_id=%s, amount=%d %s",
		paymentID, bookingID, paymentIntent.Amount, paymentIntent.Currency))

	// Complete the payment if we have payment_id
	// Use CompletePaymentFromWebhook instead of ProcessPayment to avoid creating new PaymentIntent
	if paymentID != "" {
		payment, err := h.paymentService.CompletePaymentFromWebhook(c.Request.Context(), paymentID, paymentIntent.ID)
		if err != nil {
			log.Error(fmt.Sprintf("Failed to complete payment %s: %v", paymentID, err))
			// Still return 200 to acknowledge receipt
		} else {
			log.Info(fmt.Sprintf("Payment %s completed successfully, status: %s", paymentID, payment.Status))
		}
	}

	c.JSON(http.StatusOK, gin.H{"received": true})
}

// handlePaymentIntentFailed handles failed payment
func (h *WebhookHandler) handlePaymentIntentFailed(c *gin.Context, event stripe.Event) {
	log := logger.Get()

	var paymentIntent stripe.PaymentIntent
	if err := json.Unmarshal(event.Data.Raw, &paymentIntent); err != nil {
		log.Error(fmt.Sprintf("Failed to parse payment_intent.payment_failed: %v", err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse event data"})
		return
	}

	paymentID := paymentIntent.Metadata["payment_id"]
	bookingID := paymentIntent.Metadata["booking_id"]

	failureCode := "PAYMENT_FAILED"
	failureMessage := "Payment failed"
	if paymentIntent.LastPaymentError != nil {
		failureMessage = paymentIntent.LastPaymentError.Msg
		if paymentIntent.LastPaymentError.Code != "" {
			failureCode = string(paymentIntent.LastPaymentError.Code)
		}
	}

	log.Warn(fmt.Sprintf("Payment failed: payment_id=%s, booking_id=%s, reason=%s",
		paymentID, bookingID, failureMessage))

	// Mark the payment as failed if we have payment_id
	if paymentID != "" {
		_, err := h.paymentService.FailPaymentFromWebhook(c.Request.Context(), paymentID, failureCode, failureMessage)
		if err != nil {
			log.Error(fmt.Sprintf("Failed to mark payment %s as failed: %v", paymentID, err))
		}
	}

	// Trigger seat release via Kafka event to booking-service
	if bookingID != "" {
		h.publishSeatReleaseEvent(c.Request.Context(), bookingID, paymentID, dto.SeatReleaseReasonPaymentFailed, failureCode, failureMessage)
	}

	c.JSON(http.StatusOK, gin.H{"received": true})
}

// handlePaymentIntentCanceled handles canceled payment
func (h *WebhookHandler) handlePaymentIntentCanceled(c *gin.Context, event stripe.Event) {
	log := logger.Get()

	var paymentIntent stripe.PaymentIntent
	if err := json.Unmarshal(event.Data.Raw, &paymentIntent); err != nil {
		log.Error(fmt.Sprintf("Failed to parse payment_intent.canceled: %v", err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse event data"})
		return
	}

	paymentID := paymentIntent.Metadata["payment_id"]
	bookingID := paymentIntent.Metadata["booking_id"]

	log.Info(fmt.Sprintf("Payment canceled: payment_id=%s, booking_id=%s", paymentID, bookingID))

	// Cancel the payment if we have payment_id
	if paymentID != "" {
		_, err := h.paymentService.CancelPayment(c.Request.Context(), paymentID)
		if err != nil {
			log.Error(fmt.Sprintf("Failed to cancel payment %s: %v", paymentID, err))
		}
	}

	// Trigger seat release via Kafka event to booking-service
	if bookingID != "" {
		h.publishSeatReleaseEvent(c.Request.Context(), bookingID, paymentID, dto.SeatReleaseReasonPaymentCanceled, "PAYMENT_CANCELED", "Payment was canceled")
	}

	c.JSON(http.StatusOK, gin.H{"received": true})
}

// handleChargeRefunded handles refunded charge
func (h *WebhookHandler) handleChargeRefunded(c *gin.Context, event stripe.Event) {
	log := logger.Get()

	var charge stripe.Charge
	if err := json.Unmarshal(event.Data.Raw, &charge); err != nil {
		log.Error(fmt.Sprintf("Failed to parse charge.refunded: %v", err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse event data"})
		return
	}

	paymentID := charge.Metadata["payment_id"]
	bookingID := charge.Metadata["booking_id"]

	log.Info(fmt.Sprintf("Charge refunded: payment_id=%s, booking_id=%s, amount_refunded=%d",
		paymentID, bookingID, charge.AmountRefunded))

	// Refund the payment if we have payment_id
	if paymentID != "" {
		_, err := h.paymentService.RefundPayment(c.Request.Context(), paymentID, "stripe_webhook_refund")
		if err != nil {
			log.Error(fmt.Sprintf("Failed to refund payment %s: %v", paymentID, err))
		}
	}

	// Trigger seat release via Kafka event to booking-service
	if bookingID != "" {
		h.publishSeatReleaseEvent(c.Request.Context(), bookingID, paymentID, dto.SeatReleaseReasonPaymentRefunded, "PAYMENT_REFUNDED", "Payment was refunded")
	}

	c.JSON(http.StatusOK, gin.H{"received": true})
}

// publishSeatReleaseEvent publishes a seat release event to Kafka
func (h *WebhookHandler) publishSeatReleaseEvent(ctx context.Context, bookingID, paymentID string, reason dto.SeatReleaseReason, failureCode, message string) {
	log := logger.Get()

	if h.kafkaProducer == nil {
		log.Warn("Kafka producer not configured, skipping seat release event")
		return
	}

	event := &dto.SeatReleaseEvent{
		EventType:   "seat_release",
		BookingID:   bookingID,
		PaymentID:   paymentID,
		Reason:      reason,
		FailureCode: failureCode,
		Message:     message,
		Timestamp:   time.Now().UTC(),
	}

	if err := h.kafkaProducer.ProduceJSON(ctx, dto.TopicSeatRelease, event.Key(), event, nil); err != nil {
		log.Error(fmt.Sprintf("Failed to publish seat release event: %v", err))
		return
	}

	log.Info(fmt.Sprintf("Published seat release event: booking_id=%s, reason=%s", bookingID, reason))
}
