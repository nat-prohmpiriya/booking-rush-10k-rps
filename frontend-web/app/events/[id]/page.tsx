"use client"

import { useState } from "react"
import { useParams, notFound } from "next/navigation"
import { EventHero } from "@/components/event-detail/event-hero"
import { EventInfo } from "@/components/event-detail/event-info"
import { TicketSelector } from "@/components/event-detail/ticket-selector"
import { StickyCheckout } from "@/components/event-detail/sticky-checkout"
import { CountdownTimer } from "@/components/event-detail/countdown-timer"
import { Header } from "@/components/header"
import { useEventDetail, useBookingSummary, type TicketZoneDisplay } from "@/hooks/use-events"
import { Skeleton } from "@/components/ui/skeleton"
import { Button } from "@/components/ui/button"

export default function EventDetailPage() {
  const params = useParams()
  const eventId = params.id as string

  const { event, shows, zones, selectedShow, isLoading, error, setSelectedShow } = useEventDetail(eventId)
  const { bookingSummary } = useBookingSummary(eventId)

  const [selectedTickets, setSelectedTickets] = useState<Record<string, number>>({})

  // Show loading state
  if (isLoading) {
    return (
      <div className="min-h-screen bg-[#0a0a0a] text-white">
        <Header />
        <div className="h-[60vh] relative">
          <Skeleton className="w-full h-full" />
        </div>
        <div className="container mx-auto px-4 pb-32">
          <div className="relative -mt-32 z-10">
            <div className="max-w-4xl mx-auto space-y-8">
              <Skeleton className="h-16 w-3/4" />
              <Skeleton className="h-8 w-1/2" />
              <Skeleton className="h-32 w-full" />
            </div>
          </div>
        </div>
      </div>
    )
  }

  // If no event found, show 404
  if (!event || error) {
    notFound()
  }

  const handleTicketChange = (zoneId: string, quantity: number) => {
    setSelectedTickets((prev) => {
      if (quantity === 0) {
        const { [zoneId]: _, ...rest } = prev
        return rest
      }
      return { ...prev, [zoneId]: quantity }
    })
  }

  const getTotalPrice = () => {
    return Object.entries(selectedTickets).reduce((total, [zoneId, quantity]) => {
      const zone = zones.find((z) => z.id === zoneId)
      return total + (zone?.price || 0) * quantity
    }, 0)
  }

  const getTotalTickets = () => {
    return Object.values(selectedTickets).reduce((sum, qty) => sum + qty, 0)
  }

  // Format show time for display
  const formatShowTime = () => {
    if (!selectedShow) return "TBA"
    const startTime = new Date(selectedShow.start_time)
    const endTime = new Date(selectedShow.end_time)
    return `${startTime.toLocaleTimeString("en-US", { hour: "numeric", minute: "2-digit", hour12: true })} - ${endTime.toLocaleTimeString("en-US", { hour: "numeric", minute: "2-digit", hour12: true })}`
  }

  // Format doors open time
  const formatDoorsOpen = () => {
    if (!selectedShow?.doors_open_at) {
      if (!selectedShow) return "TBA"
      // Default to 1 hour before start
      const startTime = new Date(selectedShow.start_time)
      const doorsOpen = new Date(startTime.getTime() - 60 * 60 * 1000)
      return doorsOpen.toLocaleTimeString("en-US", { hour: "numeric", minute: "2-digit", hour12: true })
    }
    const doorsOpen = new Date(selectedShow.doors_open_at)
    return doorsOpen.toLocaleTimeString("en-US", { hour: "numeric", minute: "2-digit", hour12: true })
  }

  // Convert zones to TicketSelector format
  const ticketZones = zones.map((zone: TicketZoneDisplay) => ({
    id: zone.id,
    name: zone.name,
    price: zone.price,
    available: zone.available,
    soldOut: zone.soldOut,
    maxPerOrder: zone.maxPerOrder,
    minPerOrder: zone.minPerOrder,
  }))

  // Check if event has ended (show date has passed)
  const getShowDate = (): Date | undefined => {
    if (selectedShow?.show_date) {
      return new Date(selectedShow.show_date)
    }
    return undefined
  }

  const isEventEnded = (): boolean => {
    const showDate = getShowDate()
    if (!showDate) return false
    const now = new Date()
    // Add 24 hours to show date to account for full day
    const showEndOfDay = new Date(showDate.getTime() + (24 * 60 * 60 * 1000))
    return now > showEndOfDay
  }

  return (
    <div className="min-h-screen bg-[#0a0a0a] text-white">
      <Header />
      <EventHero image={event.heroImage || event.image} />

      <div className="container mx-auto px-4 pb-32">
        <div className="relative -mt-32 z-10">
          <div className="max-w-4xl mx-auto space-y-8">
            <div>
              <h1 className="text-5xl md:text-6xl font-bold mb-4 text-balance">{event.title}</h1>
              <p className="text-xl text-[#d4af37] text-pretty">{event.subtitle}</p>
            </div>

            <CountdownTimer
              targetDate={selectedShow?.sale_start_at ? new Date(selectedShow.sale_start_at) : undefined}
              saleEndDate={selectedShow?.sale_end_at ? new Date(selectedShow.sale_end_at) : undefined}
              showDate={getShowDate()}
              showStatus={selectedShow?.status}
            />

            {/* Show selector if multiple shows */}
            {shows.length > 1 && (
              <div className="bg-zinc-900/50 p-6 rounded-xl">
                <h3 className="text-lg font-semibold mb-4">Select Show</h3>
                <div className="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 gap-4">
                  {shows.map((show) => (
                    <Button
                      key={show.id}
                      variant="outline"
                      onClick={() => {
                        setSelectedShow(show)
                        setSelectedTickets({}) // Reset ticket selection
                      }}
                      className={`h-auto p-4 flex flex-col items-start justify-start transition-all ${
                        selectedShow?.id === show.id
                          ? "border-[#d4af37] bg-[#d4af37]/10 border-2"
                          : "border-zinc-700 hover:border-zinc-500"
                      }`}
                    >
                      <div className="text-sm text-zinc-400">{show.show_date}</div>
                      <div className="font-medium">{show.name}</div>
                      <div className="text-sm text-zinc-400">
                        {new Date(show.start_time).toLocaleTimeString("en-US", { hour: "numeric", minute: "2-digit", hour12: true })}
                      </div>
                    </Button>
                  ))}
                </div>
              </div>
            )}

            <EventInfo
              date={selectedShow ? new Date(selectedShow.show_date).toLocaleDateString("en-US", { weekday: "long", month: "long", day: "numeric" }) : (event.fullDate?.split(",").slice(0, 2).join(",") || event.date)}
              year={selectedShow ? new Date(selectedShow.show_date).getFullYear().toString() : new Date().getFullYear().toString()}
              time={formatShowTime()}
              doorsOpen={formatDoorsOpen()}
              venue={event.venue.split(",")[0]}
              location={event.venue.includes(",") ? event.venue.split(",")[1].trim() : (event.city || event.venue)}
            />

            {zones.length > 0 ? (
              <TicketSelector
                zones={ticketZones}
                selectedTickets={selectedTickets}
                onTicketChange={handleTicketChange}
                bookingSummary={bookingSummary ? {
                  bookedCount: bookingSummary.booked_count,
                  maxAllowed: bookingSummary.max_allowed,
                  remainingSlots: bookingSummary.remaining_slots,
                } : null}
              />
            ) : (
              <div className="bg-zinc-900/50 p-6 rounded-xl text-center">
                <p className="text-zinc-400">No ticket zones available for this show.</p>
              </div>
            )}
          </div>
        </div>
      </div>

      <StickyCheckout
        eventId={eventId}
        showId={selectedShow?.id}
        selectedTickets={selectedTickets}
        totalPrice={getTotalPrice()}
        totalTickets={getTotalTickets()}
        isEventEnded={isEventEnded()}
        showStatus={selectedShow?.status}
      />
    </div>
  )
}
