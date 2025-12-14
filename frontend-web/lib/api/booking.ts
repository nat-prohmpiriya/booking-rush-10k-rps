import { apiClient } from "./client"
import type {
  ReserveSeatsRequest,
  ReserveSeatsResponse,
  ConfirmBookingRequest,
  ConfirmBookingResponse,
  BookingResponse,
  BookingSummaryResponse,
  CreatePaymentRequest,
  PaymentResponse,
  CreatePaymentIntentRequest,
  PaymentIntentResponse,
  ConfirmPaymentIntentRequest,
  JoinQueueRequest,
  JoinQueueResponse,
  QueuePositionResponse,
  LeaveQueueRequest,
  LeaveQueueResponse,
} from "./types"

export const bookingApi = {
  async reserveSeats(data: ReserveSeatsRequest, queuePass?: string): Promise<ReserveSeatsResponse> {
    const idempotencyKey = `reserve-${data.event_id}-${data.zone_id}-${Date.now()}`
    const headers: Record<string, string> = { "X-Idempotency-Key": idempotencyKey }
    if (queuePass) {
      headers["X-Queue-Pass"] = queuePass
    }
    return apiClient.post<ReserveSeatsResponse>("/bookings/reserve", data, { headers })
  },

  async confirmBooking(bookingId: string, data?: ConfirmBookingRequest): Promise<ConfirmBookingResponse> {
    const idempotencyKey = `confirm-${bookingId}-${Date.now()}`
    return apiClient.post<ConfirmBookingResponse>(`/bookings/${bookingId}/confirm`, data || {}, {
      headers: { "X-Idempotency-Key": idempotencyKey }
    })
  },

  async releaseBooking(bookingId: string): Promise<{ booking_id: string; status: string; message: string }> {
    const idempotencyKey = `release-${bookingId}-${Date.now()}`
    return apiClient.delete(`/bookings/${bookingId}`, {
      headers: { "X-Idempotency-Key": idempotencyKey }
    })
  },

  async getBooking(bookingId: string): Promise<BookingResponse> {
    return apiClient.get<BookingResponse>(`/bookings/${bookingId}`)
  },

  async listUserBookings(): Promise<BookingResponse[]> {
    // Backend returns PaginatedResponse: { data, page, page_size, ... }
    const response = await apiClient.get<{ data: BookingResponse[]; page: number; page_size: number }>("/bookings")
    // Handle both paginated and array responses
    if (Array.isArray(response)) {
      return response
    }
    return response.data || []
  },

  async getBookingSummary(eventId: string): Promise<BookingSummaryResponse> {
    return apiClient.get<BookingSummaryResponse>(`/bookings/summary?event_id=${eventId}`)
  },
}

export const paymentApi = {
  async createPayment(data: CreatePaymentRequest): Promise<PaymentResponse> {
    const idempotencyKey = `payment-${data.booking_id}-${Date.now()}`
    return apiClient.post<PaymentResponse>("/payments", data, {
      requireAuth: true,
      headers: { "X-Idempotency-Key": idempotencyKey }
    })
  },

  async getPayment(paymentId: string): Promise<PaymentResponse> {
    return apiClient.get<PaymentResponse>(`/payments/${paymentId}`, { requireAuth: true })
  },

  // Stripe PaymentIntent APIs
  async createPaymentIntent(data: CreatePaymentIntentRequest): Promise<PaymentIntentResponse> {
    const idempotencyKey = `payment-intent-${data.booking_id}-${Date.now()}`
    return apiClient.post<PaymentIntentResponse>("/payments/intent", data, {
      requireAuth: true,
      headers: { "X-Idempotency-Key": idempotencyKey }
    })
  },

  async confirmPaymentIntent(data: ConfirmPaymentIntentRequest): Promise<{ payment_id: string; status: string; payment_intent_id: string; stripe_status: string }> {
    const idempotencyKey = `payment-confirm-${data.payment_intent_id}-${Date.now()}`
    return apiClient.post("/payments/intent/confirm", data, {
      requireAuth: true,
      headers: { "X-Idempotency-Key": idempotencyKey }
    })
  },
}

export const queueApi = {
  async joinQueue(data: JoinQueueRequest): Promise<JoinQueueResponse> {
    // Generate idempotency key from user + event to prevent duplicate joins
    const idempotencyKey = `queue-join-${data.event_id}-${Date.now()}`
    return apiClient.post<JoinQueueResponse>("/queue/join", data, {
      headers: { "X-Idempotency-Key": idempotencyKey }
    })
  },

  async getPosition(eventId: string): Promise<QueuePositionResponse> {
    return apiClient.get<QueuePositionResponse>(`/queue/position/${eventId}`)
  },

  async leaveQueue(data: LeaveQueueRequest): Promise<LeaveQueueResponse> {
    const idempotencyKey = `queue-leave-${data.event_id}-${Date.now()}`
    return apiClient.delete<LeaveQueueResponse>("/queue/leave", {
      data,
      headers: { "X-Idempotency-Key": idempotencyKey }
    })
  },
}
