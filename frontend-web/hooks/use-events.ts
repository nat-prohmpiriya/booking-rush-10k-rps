"use client"

import { useState, useEffect, useCallback } from "react"
import { eventsApi, showsApi, zonesApi } from "@/lib/api/events"
import type { EventResponse, EventListFilter, ShowResponse, ShowZoneResponse } from "@/lib/api/types"

interface UseEventsReturn {
  events: EventDisplay[]
  isLoading: boolean
  error: string | null
  refetch: () => Promise<void>
  total: number
}

export interface EventDisplay {
  id: string | number
  title: string
  subtitle?: string
  venue: string
  date: string
  fullDate?: string
  time?: string
  image: string
  heroImage?: string
  price: number
  status?: string
  saleStatus?: string // Aggregated from shows: scheduled, on_sale, sold_out, cancelled, completed
  city?: string
  country?: string
  bookingStartAt?: string
  bookingEndAt?: string
  description?: string
}

function mapApiEventToDisplay(event: EventResponse): EventDisplay {
  // Use booking_start_at if available, otherwise use created_at
  const dateStr = event.booking_start_at || event.created_at
  const startDate = dateStr ? new Date(dateStr) : new Date()

  return {
    id: event.id,
    title: event.name,
    subtitle: event.short_description,
    venue: event.venue_name || "TBA",
    date: startDate.toLocaleDateString("en-US", { month: "short", day: "numeric" }),
    fullDate: startDate.toLocaleDateString("en-US", { weekday: "long", year: "numeric", month: "long", day: "numeric" }),
    image: event.poster_url || "/images/events/event-1.jpg",
    heroImage: event.banner_url || event.poster_url || "/images/events/event-1.jpg",
    price: event.min_price || 0,
    status: event.status,
    saleStatus: event.sale_status,
    city: event.city,
    country: event.country,
    bookingStartAt: event.booking_start_at,
    bookingEndAt: event.booking_end_at,
    description: event.description,
  }
}


export function useEvents(filter?: EventListFilter): UseEventsReturn {
  const [events, setEvents] = useState<EventDisplay[]>([])
  const [isLoading, setIsLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [total, setTotal] = useState(0)

  const fetchEvents = useCallback(async () => {
    setIsLoading(true)
    setError(null)
    try {
      const response = await eventsApi.list(filter)

      // Response is now { data: EventResponse[], meta: { total, ... } }
      const eventsList = response.data || []

      const mappedEvents = eventsList.map(mapApiEventToDisplay)

      setEvents(mappedEvents)
      setTotal(response.meta?.total || 0)
    } catch (err) {
      console.error("❌ Failed to fetch events from API:", err)
      setError("Failed to load events. Please try again later.")
      setEvents([])
      setTotal(0)
    } finally {
      setIsLoading(false)
    }
  }, [filter])

  useEffect(() => {
    fetchEvents()
  }, [fetchEvents])

  return {
    events,
    isLoading,
    error,
    refetch: fetchEvents,
    total,
  }
}

interface UseEventReturn {
  event: EventDisplay | null
  isLoading: boolean
  error: string | null
}

export function useEvent(id: string | number): UseEventReturn {
  const [event, setEvent] = useState<EventDisplay | null>(null)
  const [isLoading, setIsLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    async function fetchEvent() {
      setIsLoading(true)
      setError(null)
      try {
        const response = await eventsApi.getById(String(id))
        setEvent(mapApiEventToDisplay(response))
      } catch (err) {
        console.error("Failed to fetch event from API:", err)
        setError("Event not found")
      } finally {
        setIsLoading(false)
      }
    }

    fetchEvent()
  }, [id])

  return { event, isLoading, error }
}

// TicketZone interface for display
export interface TicketZoneDisplay {
  id: string
  name: string
  price: number
  available: number
  soldOut: boolean
  description?: string
  color?: string
  currency?: string
  minPerOrder?: number
  maxPerOrder?: number
}

// Hook for fetching shows by event slug
interface UseShowsReturn {
  shows: ShowResponse[]
  isLoading: boolean
  error: string | null
}

export function useShowsByEvent(eventSlug: string): UseShowsReturn {
  const [shows, setShows] = useState<ShowResponse[]>([])
  const [isLoading, setIsLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    async function fetchShows() {
      if (!eventSlug) return
      setIsLoading(true)
      setError(null)
      try {
        const response = await showsApi.listByEvent(eventSlug)
        setShows(response.data || [])
      } catch (err) {
        console.warn("Failed to fetch shows from API:", err)
        setError("Failed to load shows")
        setShows([])
      } finally {
        setIsLoading(false)
      }
    }

    fetchShows()
  }, [eventSlug])

  return { shows, isLoading, error }
}

// Hook for fetching zones by show ID
interface UseZonesReturn {
  zones: TicketZoneDisplay[]
  isLoading: boolean
  error: string | null
}

export function useZonesByShow(showId: string): UseZonesReturn {
  const [zones, setZones] = useState<TicketZoneDisplay[]>([])
  const [isLoading, setIsLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    async function fetchZones() {
      if (!showId) return
      setIsLoading(true)
      setError(null)
      try {
        // Use eventsApi.getShowZones which defaults to isActive=true for customers
        const apiZones = await eventsApi.getShowZones(showId)
        setZones(apiZones.map((zone: ShowZoneResponse): TicketZoneDisplay => ({
          id: zone.id,
          name: zone.name,
          price: zone.price,
          available: zone.available_seats,
          soldOut: zone.available_seats === 0,
          description: zone.description,
          color: zone.color,
          currency: zone.currency,
          minPerOrder: zone.min_per_order,
          maxPerOrder: zone.max_per_order,
        })))
      } catch (err) {
        console.warn("Failed to fetch zones from API:", err)
        setError("Failed to load ticket zones")
        setZones([])
      } finally {
        setIsLoading(false)
      }
    }

    fetchZones()
  }, [showId])

  return { zones, isLoading, error }
}

// Hook for fetching full event detail with shows and zones
export interface EventDetailData {
  event: EventDisplay | null
  shows: ShowResponse[]
  zones: TicketZoneDisplay[]
  selectedShow: ShowResponse | null
  isLoading: boolean
  error: string | null
}

export function useEventDetail(eventId: string): EventDetailData & {
  setSelectedShow: (show: ShowResponse | null) => void
} {
  const [event, setEvent] = useState<EventDisplay | null>(null)
  const [shows, setShows] = useState<ShowResponse[]>([])
  const [zones, setZones] = useState<TicketZoneDisplay[]>([])
  const [selectedShow, setSelectedShow] = useState<ShowResponse | null>(null)
  const [isLoading, setIsLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  // Fetch event
  useEffect(() => {
    async function fetchEventDetail() {
      if (!eventId) return
      setIsLoading(true)
      setError(null)

      try {
        // Fetch event by ID
        const eventResponse = await eventsApi.getById(eventId)
        const mappedEvent = mapApiEventToDisplay(eventResponse)
        setEvent(mappedEvent)

        // Fetch shows by event slug
        try {
          const showsResponse = await showsApi.listByEvent(eventResponse.slug)
          const showsList = showsResponse.data || []
          setShows(showsList)

          // Auto-select first show
          if (showsList.length > 0) {
            setSelectedShow(showsList[0])
          }
        } catch (showErr) {
          console.warn("Failed to fetch shows:", showErr)
          setShows([])
        }
      } catch (err) {
        console.error("Failed to fetch event from API:", err)
        setError("Event not found")
      } finally {
        setIsLoading(false)
      }
    }

    fetchEventDetail()
  }, [eventId])

  // Fetch zones when selected show changes (customer view - only active zones)
  useEffect(() => {
    async function fetchZones() {
      if (!selectedShow) {
        setZones([])
        return
      }

      try {
        // Use eventsApi.getShowZones which defaults to isActive=true for customers
        const apiZones = await eventsApi.getShowZones(selectedShow.id)
        setZones(apiZones.map((zone: ShowZoneResponse): TicketZoneDisplay => ({
          id: zone.id,
          name: zone.name,
          price: zone.price,
          available: zone.available_seats,
          soldOut: zone.available_seats === 0,
          description: zone.description,
          color: zone.color,
          currency: zone.currency,
          minPerOrder: zone.min_per_order,
          maxPerOrder: zone.max_per_order,
        })))
      } catch (err) {
        console.warn("Failed to fetch zones:", err)
        setZones([])
      }
    }

    fetchZones()
  }, [selectedShow])

  return {
    event,
    shows,
    zones,
    selectedShow,
    isLoading,
    error,
    setSelectedShow,
  }
}
