"use client"

import { Suspense, useEffect, useState } from "react"
import { useSearchParams } from "next/navigation"
import { Check, Download, Wallet, Calendar, MapPin, Ticket, AlertTriangle, Loader2 } from "lucide-react"
import { QRCodeSVG } from "qrcode.react"
import { Button } from "@/components/ui/button"
import { Card } from "@/components/ui/card"
import Link from "next/link"
import { bookingApi } from "@/lib/api/booking"
import { eventsApi } from "@/lib/api/events"
import type { BookingResponse, EventResponse, ShowResponse, ShowZoneResponse } from "@/lib/api/types"

interface BookingDetails {
  booking: BookingResponse
  event: EventResponse | null
  show: ShowResponse | null
  zone: ShowZoneResponse | null
}

// Loading fallback component
function LoadingFallback() {
  return (
    <div className="min-h-screen bg-[#0a0a0a] text-white flex items-center justify-center">
      <div className="text-center space-y-4">
        <Loader2 className="w-12 h-12 animate-spin text-[#d4af37] mx-auto" />
        <p className="text-zinc-400">Loading your booking...</p>
      </div>
    </div>
  )
}

// Main page component with Suspense wrapper
export default function BookingConfirmationPage() {
  return (
    <Suspense fallback={<LoadingFallback />}>
      <BookingConfirmationContent />
    </Suspense>
  )
}

// Content component that uses useSearchParams
function BookingConfirmationContent() {
  const searchParams = useSearchParams()
  const bookingId = searchParams.get("booking_id")

  const [showSuccess, setShowSuccess] = useState(false)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [bookingDetails, setBookingDetails] = useState<BookingDetails | null>(null)

  useEffect(() => {
    if (!bookingId) {
      setError("No booking ID provided")
      setLoading(false)
      return
    }

    const fetchBookingDetails = async () => {
      try {
        // Fetch booking
        const booking = await bookingApi.getBooking(bookingId)

        // Fetch event details
        let event: EventResponse | null = null
        let show: ShowResponse | null = null
        let zone: ShowZoneResponse | null = null

        if (booking.event_id) {
          try {
            event = await eventsApi.getEvent(booking.event_id)
          } catch (err) {
            console.error("Failed to fetch event:", err)
          }
        }

        // Try to get show and zone info if available
        if (event?.slug) {
          try {
            const shows = await eventsApi.getEventShowsBySlug(event.slug)
            // Find matching show (simplified - in real app might need show_id in booking)
            if (shows.length > 0) {
              show = shows[0]
              if (booking.zone_id) {
                const zones = await eventsApi.getShowZones(show.id)
                zone = zones.find((z: ShowZoneResponse) => z.id === booking.zone_id) || null
              }
            }
          } catch (err) {
            console.error("Failed to fetch show/zone:", err)
          }
        }

        setBookingDetails({ booking, event, show, zone })
        setLoading(false)

        // Trigger success animation
        setTimeout(() => setShowSuccess(true), 100)
      } catch (err) {
        console.error("Failed to fetch booking:", err)
        setError("Failed to load booking details")
        setLoading(false)
      }
    }

    fetchBookingDetails()
  }, [bookingId])

  // Format date
  const formatDate = (dateString: string) => {
    const date = new Date(dateString)
    return date.toLocaleDateString("en-US", {
      weekday: "long",
      month: "long",
      day: "numeric",
      year: "numeric",
    })
  }

  // Format time
  const formatTime = (dateString: string) => {
    const date = new Date(dateString)
    return date.toLocaleTimeString("en-US", {
      hour: "numeric",
      minute: "2-digit",
      hour12: true,
    })
  }

  // Loading state
  if (loading) {
    return (
      <div className="min-h-screen bg-[#0a0a0a] text-white flex items-center justify-center">
        <div className="text-center space-y-4">
          <Loader2 className="w-12 h-12 animate-spin text-[#d4af37] mx-auto" />
          <p className="text-zinc-400">Loading your booking...</p>
        </div>
      </div>
    )
  }

  // Error state
  if (error || !bookingDetails) {
    return (
      <div className="min-h-screen bg-[#0a0a0a] text-white flex items-center justify-center p-4">
        <Card className="bg-zinc-900/50 border-red-800 p-8 max-w-md text-center">
          <div className="w-16 h-16 rounded-full border-2 border-red-500 flex items-center justify-center mx-auto mb-4">
            <AlertTriangle className="w-8 h-8 text-red-500" />
          </div>
          <h1 className="text-2xl font-bold text-white mb-2">Booking Not Found</h1>
          <p className="text-zinc-400 mb-6">{error || "Unable to load booking details"}</p>
          <Link href="/">
            <Button className="bg-[#d4af37] hover:bg-[#c19d2f] text-[#0a0a0a]">Back to Home</Button>
          </Link>
        </Card>
      </div>
    )
  }

  const { booking, event, show, zone } = bookingDetails

  return (
    <div className="min-h-screen bg-[#0a0a0a] text-white flex items-center justify-center p-4">
      <div className="max-w-2xl w-full space-y-8 py-12">
        {/* Success Animation */}
        <div className="flex flex-col items-center space-y-6">
          <div
            className={`relative transition-all duration-700 ${
              showSuccess ? "scale-100 opacity-100" : "scale-50 opacity-0"
            }`}
          >
            <div className="w-24 h-24 rounded-full bg-[#d4af37]/10 flex items-center justify-center relative">
              {/* Gold glow effect */}
              <div className="absolute inset-0 rounded-full bg-[#d4af37] opacity-20 blur-xl animate-pulse" />
              <div className="relative w-16 h-16 rounded-full bg-[#d4af37] flex items-center justify-center">
                <Check className="w-10 h-10 text-[#0a0a0a] stroke-[3]" />
              </div>
            </div>
          </div>

          <div className="text-center space-y-2">
            <h1 className="text-4xl md:text-5xl font-bold text-balance">Booking Confirmed!</h1>
            <p className="text-lg text-zinc-400">Your tickets have been sent to your email</p>
          </div>
        </div>

        {/* E-Ticket Card */}
        <Card className="bg-zinc-900/50 border-zinc-800 backdrop-blur-sm overflow-hidden">
          <div className="p-6 md:p-8 space-y-6">
            {/* QR Code Section */}
            <div className="flex flex-col md:flex-row gap-6 items-start md:items-center">
              <div className="shrink-0">
                <div className="w-32 h-32 bg-white rounded-lg p-2 shadow-lg shadow-[#d4af37]/20">
                  <QRCodeSVG
                    value={booking.id}
                    size={112}
                    bgColor="white"
                    fgColor="black"
                    level="M"
                  />
                </div>
              </div>

              <div className="flex-1 space-y-4">
                {/* Booking ID */}
                <div>
                  <p className="text-sm text-zinc-500 uppercase tracking-wider">Booking ID</p>
                  <p className="text-lg font-bold text-[#d4af37] font-mono tracking-wide break-all">{booking.id}</p>
                </div>

                {/* Event Name */}
                <div>
                  <h2 className="text-2xl font-semibold text-balance">{event?.name || "Event"}</h2>
                </div>
              </div>
            </div>

            {/* Divider */}
            <div className="border-t border-zinc-800" />

            {/* Event Details */}
            <div className="grid md:grid-cols-2 gap-6">
              {/* Date & Time */}
              <div className="flex gap-3">
                <div className="shrink-0">
                  <div className="w-10 h-10 rounded-lg bg-[#d4af37]/10 flex items-center justify-center">
                    <Calendar className="w-5 h-5 text-[#d4af37]" />
                  </div>
                </div>
                <div>
                  <p className="text-sm text-zinc-500">Date & Time</p>
                  <p className="font-medium">
                    {show?.show_date ? formatDate(show.show_date) : "Date TBA"}
                  </p>
                  <p className="text-sm text-zinc-400">
                    {show?.start_time ? formatTime(show.start_time) : "Time TBA"}
                  </p>
                </div>
              </div>

              {/* Venue */}
              <div className="flex gap-3">
                <div className="shrink-0">
                  <div className="w-10 h-10 rounded-lg bg-[#d4af37]/10 flex items-center justify-center">
                    <MapPin className="w-5 h-5 text-[#d4af37]" />
                  </div>
                </div>
                <div>
                  <p className="text-sm text-zinc-500">Venue</p>
                  <p className="font-medium">{event?.venue_name || "Venue TBA"}</p>
                  <p className="text-sm text-zinc-400">{event?.venue_address || ""}</p>
                </div>
              </div>

              {/* Zone */}
              <div className="flex gap-3">
                <div className="shrink-0">
                  <div className="w-10 h-10 rounded-lg bg-[#d4af37]/10 flex items-center justify-center">
                    <Ticket className="w-5 h-5 text-[#d4af37]" />
                  </div>
                </div>
                <div>
                  <p className="text-sm text-zinc-500">Zone</p>
                  <p className="font-medium">{zone?.name || "General Admission"}</p>
                </div>
              </div>

              {/* Quantity & Price */}
              <div className="flex gap-3">
                <div className="shrink-0">
                  <div className="w-10 h-10 rounded-lg bg-[#d4af37]/10 flex items-center justify-center">
                    <span className="text-[#d4af37] font-bold text-lg">#</span>
                  </div>
                </div>
                <div>
                  <p className="text-sm text-zinc-500">Tickets</p>
                  <p className="font-medium">{booking.quantity} ticket(s)</p>
                  <p className="text-sm text-zinc-400">Total: ฿{booking.total_price.toLocaleString()}</p>
                </div>
              </div>
            </div>

            {/* Divider */}
            <div className="border-t border-zinc-800" />

            {/* Barcode */}
            <div className="space-y-2">
              <p className="text-sm text-zinc-500 text-center">Scan at entry</p>
              <div className="flex justify-center">
                <svg
                  width="280"
                  height="60"
                  viewBox="0 0 280 60"
                  className="w-full max-w-sm"
                  xmlns="http://www.w3.org/2000/svg"
                >
                  <rect width="280" height="60" fill="white" rx="4" />
                  <g fill="black">
                    <rect x="10" y="10" width="3" height="40" />
                    <rect x="16" y="10" width="2" height="40" />
                    <rect x="22" y="10" width="4" height="40" />
                    <rect x="29" y="10" width="2" height="40" />
                    <rect x="34" y="10" width="5" height="40" />
                    <rect x="42" y="10" width="2" height="40" />
                    <rect x="47" y="10" width="3" height="40" />
                    <rect x="53" y="10" width="2" height="40" />
                    <rect x="58" y="10" width="4" height="40" />
                    <rect x="65" y="10" width="3" height="40" />
                    <rect x="71" y="10" width="2" height="40" />
                    <rect x="76" y="10" width="5" height="40" />
                    <rect x="84" y="10" width="2" height="40" />
                    <rect x="89" y="10" width="3" height="40" />
                    <rect x="95" y="10" width="4" height="40" />
                    <rect x="102" y="10" width="2" height="40" />
                    <rect x="107" y="10" width="3" height="40" />
                    <rect x="113" y="10" width="5" height="40" />
                    <rect x="121" y="10" width="2" height="40" />
                    <rect x="126" y="10" width="4" height="40" />
                    <rect x="133" y="10" width="2" height="40" />
                    <rect x="138" y="10" width="3" height="40" />
                    <rect x="144" y="10" width="2" height="40" />
                    <rect x="149" y="10" width="5" height="40" />
                    <rect x="157" y="10" width="3" height="40" />
                    <rect x="163" y="10" width="2" height="40" />
                    <rect x="168" y="10" width="4" height="40" />
                    <rect x="175" y="10" width="2" height="40" />
                    <rect x="180" y="10" width="5" height="40" />
                    <rect x="188" y="10" width="2" height="40" />
                    <rect x="193" y="10" width="3" height="40" />
                    <rect x="199" y="10" width="4" height="40" />
                    <rect x="206" y="10" width="2" height="40" />
                    <rect x="211" y="10" width="3" height="40" />
                    <rect x="217" y="10" width="2" height="40" />
                    <rect x="222" y="10" width="5" height="40" />
                    <rect x="230" y="10" width="3" height="40" />
                    <rect x="236" y="10" width="2" height="40" />
                    <rect x="241" y="10" width="4" height="40" />
                    <rect x="248" y="10" width="2" height="40" />
                    <rect x="253" y="10" width="5" height="40" />
                    <rect x="261" y="10" width="3" height="40" />
                    <rect x="267" y="10" width="2" height="40" />
                  </g>
                </svg>
              </div>
              <p className="text-xs text-zinc-500 text-center font-mono">{booking.id}</p>
            </div>
          </div>
        </Card>

        {/* Action Buttons */}
        <div className="flex flex-col sm:flex-row gap-4">
          <Button className="flex-1 h-12 bg-[#d4af37] hover:bg-[#c19d2f] text-[#0a0a0a] font-semibold" size="lg">
            <Download className="w-5 h-5 mr-2" />
            Download E-Ticket
          </Button>
          <Button
            variant="outline"
            className="flex-1 h-12 border-zinc-700 hover:bg-zinc-800 hover:text-white bg-transparent"
            size="lg"
          >
            <Wallet className="w-5 h-5 mr-2" />
            Add to Wallet
          </Button>
        </div>

        {/* View My Bookings Link */}
        <div className="text-center space-y-2">
          <Link
            href="/my-bookings"
            className="text-[#d4af37] hover:text-[#c19d2f] font-medium inline-flex items-center gap-2 transition-colors"
          >
            View My Bookings
            <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5l7 7-7 7" />
            </svg>
          </Link>
          <br />
          <Link
            href="/"
            className="text-zinc-400 hover:text-zinc-300 text-sm inline-flex items-center gap-2 transition-colors"
          >
            Back to Home
          </Link>
        </div>
      </div>
    </div>
  )
}
