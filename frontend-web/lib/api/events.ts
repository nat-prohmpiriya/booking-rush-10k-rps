import { apiClient } from "./client"
import type {
  EventResponse,
  EventListResponse,
  EventListFilter,
  ShowResponse,
  ShowListResponse,
  ShowZoneResponse,
  ShowZoneListResponse,
} from "./types"

// Update Event Request type
export interface UpdateEventRequest {
  name?: string
  description?: string
  short_description?: string
  poster_url?: string
  banner_url?: string
  venue_name?: string
  venue_address?: string
  city?: string
  country?: string
  max_tickets_per_user?: number
  booking_start_at?: string
  booking_end_at?: string
  status?: string // draft, published, cancelled, completed
  is_featured?: boolean
  is_public?: boolean
}

// Update Show Request type
export interface UpdateShowRequest {
  name?: string
  show_date?: string
  start_time?: string
  end_time?: string
  doors_open_at?: string
  status?: string
  sale_start_at?: string
  sale_end_at?: string
}

// Update Zone Request type
export interface UpdateZoneRequest {
  name?: string
  description?: string
  color?: string
  price?: number
  total_seats?: number
  min_per_order?: number
  max_per_order?: number
  is_active?: boolean
  sort_order?: number
  sale_start_at?: string
  sale_end_at?: string
}

export const eventsApi = {
  // List published events (public)
  async list(filter?: EventListFilter): Promise<EventListResponse> {
    const params = new URLSearchParams()
    if (filter?.status) params.append("status", filter.status)
    if (filter?.venue_id) params.append("venue_id", filter.venue_id)
    if (filter?.search) params.append("search", filter.search)
    if (filter?.limit) params.append("limit", filter.limit.toString())
    if (filter?.offset) params.append("offset", filter.offset.toString())

    const queryString = params.toString()
    const endpoint = queryString ? `/events?${queryString}` : "/events"
    return apiClient.get<EventListResponse>(endpoint)
  },

  // List events owned by current user (organizer)
  async listMyEvents(filter?: EventListFilter): Promise<EventListResponse> {
    const params = new URLSearchParams()
    if (filter?.status) params.append("status", filter.status)
    if (filter?.search) params.append("search", filter.search)
    if (filter?.limit) params.append("limit", filter.limit.toString())
    if (filter?.offset) params.append("offset", filter.offset.toString())

    const queryString = params.toString()
    const endpoint = queryString ? `/events/my?${queryString}` : "/events/my"
    return apiClient.get<EventListResponse>(endpoint)
  },

  async getBySlug(slug: string): Promise<EventResponse> {
    return apiClient.get<EventResponse>(`/events/${slug}`)
  },

  async getById(id: string): Promise<EventResponse> {
    return apiClient.get<EventResponse>(`/events/id/${id}`)
  },

  // Alias for getById - used by checkout page
  async getEvent(eventId: string): Promise<EventResponse> {
    return apiClient.get<EventResponse>(`/events/id/${eventId}`)
  },

  // Update an event
  async update(id: string, data: UpdateEventRequest): Promise<EventResponse> {
    return apiClient.put<EventResponse>(`/events/${id}`, data)
  },

  // Delete an event
  async delete(id: string): Promise<void> {
    return apiClient.delete(`/events/${id}`)
  },

  // Publish an event
  async publish(id: string): Promise<EventResponse> {
    return apiClient.post<EventResponse>(`/events/${id}/publish`)
  },

  // Get shows for an event by slug
  async getEventShowsBySlug(slug: string): Promise<ShowResponse[]> {
    const response = await apiClient.get<ShowListResponse>(`/events/${slug}/shows`)
    return response.data
  },

  // Get shows for an event by ID (fetches event first to get slug)
  async getEventShows(eventId: string): Promise<ShowResponse[]> {
    // First get the event to obtain the slug
    const event = await apiClient.get<EventResponse>(`/events/id/${eventId}`)
    // Then fetch shows using the slug
    const response = await apiClient.get<ShowListResponse>(`/events/${event.slug}/shows`)
    return response.data
  },

  // Get zones for a show (for customer - only active zones by default)
  async getShowZones(showId: string, isActive: boolean = true): Promise<ShowZoneResponse[]> {
    const params = new URLSearchParams()
    params.append("is_active", isActive.toString())
    const response = await apiClient.get<ShowZoneListResponse>(`/shows/${showId}/zones?${params.toString()}`)
    return response.data
  },
}

export const showsApi = {
  async listByEvent(eventSlug: string, limit?: number, offset?: number): Promise<ShowListResponse> {
    const params = new URLSearchParams()
    if (limit) params.append("limit", limit.toString())
    if (offset) params.append("offset", offset.toString())

    const queryString = params.toString()
    const endpoint = queryString
      ? `/events/${eventSlug}/shows?${queryString}`
      : `/events/${eventSlug}/shows`
    return apiClient.get<ShowListResponse>(endpoint)
  },

  async getById(showId: string): Promise<ShowResponse> {
    return apiClient.get<ShowResponse>(`/shows/${showId}`)
  },

  async update(showId: string, data: UpdateShowRequest): Promise<ShowResponse> {
    return apiClient.put<ShowResponse>(`/shows/${showId}`, data)
  },

  async delete(showId: string): Promise<void> {
    return apiClient.delete(`/shows/${showId}`)
  },
}

export const zonesApi = {
  // For organizer - can filter by is_active or get all (isActive = undefined)
  async listByShow(showId: string, options?: { limit?: number; offset?: number; isActive?: boolean }): Promise<ShowZoneListResponse> {
    const params = new URLSearchParams()
    if (options?.limit) params.append("limit", options.limit.toString())
    if (options?.offset) params.append("offset", options.offset.toString())
    if (options?.isActive !== undefined) params.append("is_active", options.isActive.toString())

    const queryString = params.toString()
    const endpoint = queryString
      ? `/shows/${showId}/zones?${queryString}`
      : `/shows/${showId}/zones`
    return apiClient.get<ShowZoneListResponse>(endpoint)
  },

  async getById(zoneId: string): Promise<ShowZoneResponse> {
    return apiClient.get<ShowZoneResponse>(`/zones/${zoneId}`)
  },

  async update(zoneId: string, data: UpdateZoneRequest): Promise<ShowZoneResponse> {
    return apiClient.put<ShowZoneResponse>(`/zones/${zoneId}`, data)
  },

  async delete(zoneId: string): Promise<void> {
    return apiClient.delete(`/zones/${zoneId}`)
  },
}
