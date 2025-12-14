// API Response types
export interface ApiResponse<T> {
  success: boolean
  data?: T
  message?: string
}

export interface ApiErrorDetails {
  code?: string
  message?: string
}

export interface ApiError {
  success: false
  error: ApiErrorDetails | string
  code?: string
  message?: string
}

export interface PaginationMeta {
  page: number
  per_page: number
  total: number
  total_pages: number
}

export interface PaginatedResponse<T> {
  data: T[]
  meta: PaginationMeta
}

// Auth types
export interface LoginRequest {
  email: string
  password: string
}

export interface RegisterRequest {
  email: string
  password: string
  name: string
}

export interface RefreshTokenRequest {
  refresh_token: string
}

export interface AuthResponse {
  access_token: string
  refresh_token: string
  expires_in: number
  user: UserResponse
}

export interface UserResponse {
  id: string
  email: string
  name: string
  role: string
  created_at: string
}

// Event types
export interface EventResponse {
  id: string
  tenant_id: string
  organizer_id: string
  category_id?: string
  name: string
  slug: string
  description: string
  short_description: string
  poster_url: string
  banner_url: string
  gallery: string[]
  venue_name: string
  venue_address: string
  city: string
  country: string
  latitude?: number
  longitude?: number
  max_tickets_per_user: number
  booking_start_at?: string
  booking_end_at?: string
  status: string
  sale_status: string // Aggregated from shows: scheduled, on_sale, sold_out, cancelled, completed
  is_featured: boolean
  is_public: boolean
  meta_title: string
  meta_description: string
  min_price: number
  published_at?: string
  created_at: string
  updated_at: string
}

export interface EventListResponse extends PaginatedResponse<EventResponse> {}

export interface EventListFilter {
  status?: string
  venue_id?: string
  search?: string
  limit?: number
  offset?: number
}

// Show types
export interface ShowResponse {
  id: string
  event_id: string
  name: string
  show_date: string
  start_time: string
  end_time: string
  doors_open_at?: string
  status: string
  sale_start_at?: string
  sale_end_at?: string
  total_capacity: number
  reserved_count: number
  sold_count: number
  created_at: string
  updated_at: string
}

export interface ShowListResponse extends PaginatedResponse<ShowResponse> {}

// Zone types
export interface ShowZoneResponse {
  id: string
  show_id: string
  name: string
  description: string
  color: string
  price: number
  currency: string
  total_seats: number
  available_seats: number
  reserved_seats: number
  sold_seats: number
  min_per_order: number
  max_per_order: number
  is_active: boolean
  sort_order: number
  sale_start_at?: string
  sale_end_at?: string
  created_at: string
  updated_at: string
}

export interface ShowZoneListResponse extends PaginatedResponse<ShowZoneResponse> {}

// Booking types
export interface ReserveSeatsRequest {
  event_id: string
  zone_id: string
  show_id?: string
  quantity: number
  unit_price?: number
}

export interface ReserveSeatsResponse {
  booking_id: string
  status: string
  expires_at: string
  total_price: number
}

export interface ConfirmBookingRequest {
  payment_id?: string
}

export interface ConfirmBookingResponse {
  booking_id: string
  status: string
  confirmed_at: string
  confirmation_code?: string
}

export interface BookingResponse {
  id: string
  user_id: string
  event_id: string
  zone_id: string
  quantity: number
  status: string
  total_price: number
  payment_id?: string
  reserved_at: string
  confirmed_at?: string
  expires_at: string
}

export interface BookingSummaryResponse {
  user_id: string
  event_id: string
  booked_count: number
  max_allowed: number
  remaining_slots: number
}

// Payment types
export interface CreatePaymentRequest {
  booking_id: string
  payment_method: string
  amount: number
}

export interface PaymentResponse {
  id: string
  booking_id: string
  amount: number
  status: string
  payment_method: string
  created_at: string
}

// Stripe PaymentIntent types
export interface CreatePaymentIntentRequest {
  booking_id: string
  amount: number
  currency?: string
}

export interface PaymentIntentResponse {
  payment_id: string
  client_secret: string
  payment_intent_id: string
  amount: number
  currency: string
  status: string
}

export interface ConfirmPaymentIntentRequest {
  payment_id: string
  payment_intent_id: string
}

// Queue types
export interface JoinQueueRequest {
  event_id: string
}

export interface JoinQueueResponse {
  position: number
  token: string
  estimated_wait_seconds: number
  joined_at: string
  expires_at: string
  message?: string
}

export interface QueuePositionResponse {
  position: number
  total_in_queue: number
  estimated_wait_seconds: number
  is_ready: boolean
  expires_at?: string
  queue_pass?: string
  queue_pass_expires_at?: string
}

export interface QueueStatusResponse {
  event_id: string
  total_in_queue: number
  is_open: boolean
}

export interface LeaveQueueRequest {
  event_id: string
  token: string
}

export interface LeaveQueueResponse {
  success: boolean
  message: string
}
