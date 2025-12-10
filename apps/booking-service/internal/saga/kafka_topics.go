package saga

// Kafka topic names for saga commands and events
const (
	// Command topics - sent by saga orchestrator to trigger step execution
	TopicSagaReserveSeatsCommand    = "saga.booking.reserve-seats.command"
	TopicSagaProcessPaymentCommand  = "saga.booking.process-payment.command"
	TopicSagaConfirmBookingCommand  = "saga.booking.confirm-booking.command"
	TopicSagaSendNotificationCommand = "saga.booking.send-notification.command"

	// Compensation command topics - sent when step needs to be compensated
	TopicSagaReleaseSeatsCommand   = "saga.booking.release-seats.command"
	TopicSagaRefundPaymentCommand  = "saga.booking.refund-payment.command"

	// Event topics - published after step execution
	TopicSagaSeatsReservedEvent      = "saga.booking.seats-reserved.event"
	TopicSagaSeatsReleasedEvent      = "saga.booking.seats-released.event"
	TopicSagaPaymentProcessedEvent   = "saga.booking.payment-processed.event"
	TopicSagaPaymentRefundedEvent    = "saga.booking.payment-refunded.event"
	TopicSagaBookingConfirmedEvent   = "saga.booking.booking-confirmed.event"
	TopicSagaNotificationSentEvent   = "saga.booking.notification-sent.event"

	// Failure event topics
	TopicSagaSeatsReservationFailedEvent = "saga.booking.seats-reservation-failed.event"
	TopicSagaPaymentFailedEvent          = "saga.booking.payment-failed.event"
	TopicSagaBookingConfirmationFailedEvent = "saga.booking.booking-confirmation-failed.event"
	TopicSagaNotificationFailedEvent     = "saga.booking.notification-failed.event"

	// Saga lifecycle topics
	TopicSagaStartedEvent    = "saga.booking.started.event"
	TopicSagaCompletedEvent  = "saga.booking.completed.event"
	TopicSagaFailedEvent     = "saga.booking.failed.event"
	TopicSagaCompensatedEvent = "saga.booking.compensated.event"
)

// GetAllCommandTopics returns all command topics for the booking saga
func GetAllCommandTopics() []string {
	return []string{
		TopicSagaReserveSeatsCommand,
		TopicSagaProcessPaymentCommand,
		TopicSagaConfirmBookingCommand,
		TopicSagaSendNotificationCommand,
		TopicSagaReleaseSeatsCommand,
		TopicSagaRefundPaymentCommand,
	}
}

// GetAllEventTopics returns all event topics for the booking saga
func GetAllEventTopics() []string {
	return []string{
		TopicSagaSeatsReservedEvent,
		TopicSagaSeatsReleasedEvent,
		TopicSagaPaymentProcessedEvent,
		TopicSagaPaymentRefundedEvent,
		TopicSagaBookingConfirmedEvent,
		TopicSagaNotificationSentEvent,
		TopicSagaSeatsReservationFailedEvent,
		TopicSagaPaymentFailedEvent,
		TopicSagaBookingConfirmationFailedEvent,
		TopicSagaNotificationFailedEvent,
		TopicSagaStartedEvent,
		TopicSagaCompletedEvent,
		TopicSagaFailedEvent,
		TopicSagaCompensatedEvent,
	}
}

// StepToCommandTopic maps saga step names to their command topics
func StepToCommandTopic(stepName string) string {
	switch stepName {
	case StepReserveSeats:
		return TopicSagaReserveSeatsCommand
	case StepProcessPayment:
		return TopicSagaProcessPaymentCommand
	case StepConfirmBooking:
		return TopicSagaConfirmBookingCommand
	case StepSendNotification:
		return TopicSagaSendNotificationCommand
	default:
		return ""
	}
}

// StepToCompensationTopic maps saga step names to their compensation command topics
func StepToCompensationTopic(stepName string) string {
	switch stepName {
	case StepReserveSeats:
		return TopicSagaReleaseSeatsCommand
	case StepProcessPayment:
		return TopicSagaRefundPaymentCommand
	default:
		return ""
	}
}

// StepToSuccessEventTopic maps saga step names to their success event topics
func StepToSuccessEventTopic(stepName string) string {
	switch stepName {
	case StepReserveSeats:
		return TopicSagaSeatsReservedEvent
	case StepProcessPayment:
		return TopicSagaPaymentProcessedEvent
	case StepConfirmBooking:
		return TopicSagaBookingConfirmedEvent
	case StepSendNotification:
		return TopicSagaNotificationSentEvent
	default:
		return ""
	}
}

// StepToFailureEventTopic maps saga step names to their failure event topics
func StepToFailureEventTopic(stepName string) string {
	switch stepName {
	case StepReserveSeats:
		return TopicSagaSeatsReservationFailedEvent
	case StepProcessPayment:
		return TopicSagaPaymentFailedEvent
	case StepConfirmBooking:
		return TopicSagaBookingConfirmationFailedEvent
	case StepSendNotification:
		return TopicSagaNotificationFailedEvent
	default:
		return ""
	}
}
