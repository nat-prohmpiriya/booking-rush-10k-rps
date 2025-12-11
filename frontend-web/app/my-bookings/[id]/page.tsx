"use client"

import { useState, useEffect } from "react"
import { useParams, useRouter } from "next/navigation"
import Link from "next/link"
import { Header } from "@/components/header"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import { Card } from "@/components/ui/card"
import { Skeleton } from "@/components/ui/skeleton"
import { useAuth } from "@/contexts/auth-context"
import { bookingApi } from "@/lib/api/booking"
import { eventsApi, zonesApi, showsApi } from "@/lib/api/events"
import type { BookingResponse, EventResponse, ShowResponse, ShowZoneResponse } from "@/lib/api/types"
import {
  Ticket,
  Calendar,
  Clock,
  MapPin,
  ArrowLeft,
  QrCode,
  Download,
  Wallet,
  CheckCircle2,
  XCircle,
  Timer,
  AlertCircle,
  CreditCard,
  History,
  ExternalLink,
} from "lucide-react"

interface BookingDetail {
  booking: BookingResponse
  event: EventResponse | null
  show: ShowResponse | null
  zone: ShowZoneResponse | null
}

function getStatusConfig(status: string) {
  switch (status) {
    case "confirmed":
      return {
        label: "Confirmed",
        color: "bg-green-500/20 text-green-400 border-green-500/30",
        icon: CheckCircle2,
        description: "Your booking is confirmed. Show this ticket at the venue.",
      }
    case "reserved":
    case "pending":
      return {
        label: "Pending Payment",
        color: "bg-amber-500/20 text-amber-400 border-amber-500/30",
        icon: Timer,
        description: "Complete your payment to confirm this booking.",
      }
    case "completed":
      return {
        label: "Completed",
        color: "bg-blue-500/20 text-blue-400 border-blue-500/30",
        icon: CheckCircle2,
        description: "This event has ended. Thank you for attending!",
      }
    case "cancelled":
      return {
        label: "Cancelled",
        color: "bg-red-500/20 text-red-400 border-red-500/30",
        icon: XCircle,
        description: "This booking has been cancelled.",
      }
    case "expired":
      return {
        label: "Expired",
        color: "bg-zinc-500/20 text-zinc-400 border-zinc-500/30",
        icon: AlertCircle,
        description: "This booking has expired due to incomplete payment.",
      }
    default:
      return {
        label: status,
        color: "bg-zinc-500/20 text-zinc-400 border-zinc-500/30",
        icon: AlertCircle,
        description: "",
      }
  }
}

function generateReference(id: string) {
  const shortId = id.split("-")[0]?.toUpperCase() || id.slice(0, 8).toUpperCase()
  return `BK-2025-${shortId}`
}

function formatDateTime(dateString: string) {
  const date = new Date(dateString)
  return {
    date: date.toLocaleDateString("en-US", {
      weekday: "short",
      month: "short",
      day: "numeric",
      year: "numeric",
    }),
    time: date.toLocaleTimeString("en-US", {
      hour: "numeric",
      minute: "2-digit",
      hour12: true,
    }),
  }
}

function BookingDetailSkeleton() {
  return (
    <div className="space-y-6">
      <Skeleton className="h-8 w-48" />
      <Skeleton className="h-40 w-full rounded-xl" />
      <Skeleton className="h-64 w-full rounded-xl" />
      <Skeleton className="h-48 w-full rounded-xl" />
    </div>
  )
}

export default function BookingDetailPage() {
  const params = useParams()
  const router = useRouter()
  const bookingId = params.id as string
  const { isAuthenticated, isLoading: authLoading } = useAuth()

  const [bookingDetail, setBookingDetail] = useState<BookingDetail | null>(null)
  const [isLoading, setIsLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    if (!authLoading && !isAuthenticated) {
      router.push(`/login?redirect=/my-bookings/${bookingId}`)
      return
    }

    async function fetchBookingDetail() {
      if (!bookingId) return

      setIsLoading(true)
      setError(null)

      try {
        // Fetch booking
        const booking = await bookingApi.getBooking(bookingId)

        let event: EventResponse | null = null
        let show: ShowResponse | null = null
        let zone: ShowZoneResponse | null = null

        // Fetch event details
        try {
          event = await eventsApi.getById(booking.event_id)
        } catch (err) {
          console.warn("Failed to fetch event:", err)
        }

        // Fetch zone details
        try {
          zone = await zonesApi.getById(booking.zone_id)
          // If zone has show_id, fetch show details
          if (zone?.show_id) {
            try {
              show = await showsApi.getById(zone.show_id)
            } catch (err) {
              console.warn("Failed to fetch show:", err)
            }
          }
        } catch (err) {
          console.warn("Failed to fetch zone:", err)
        }

        setBookingDetail({ booking, event, show, zone })
      } catch (err) {
        console.error("Failed to fetch booking:", err)
        setError("Booking not found or you don't have permission to view it.")
      } finally {
        setIsLoading(false)
      }
    }

    if (isAuthenticated) {
      fetchBookingDetail()
    }
  }, [bookingId, isAuthenticated, authLoading, router])

  if (authLoading || isLoading) {
    return (
      <main className="min-h-screen bg-background">
        <Header />
        <div className="container mx-auto px-4 lg:px-8 pt-24 pb-16 max-w-3xl">
          <BookingDetailSkeleton />
        </div>
      </main>
    )
  }

  if (error || !bookingDetail) {
    return (
      <main className="min-h-screen bg-background">
        <Header />
        <div className="container mx-auto px-4 lg:px-8 pt-24 pb-16 max-w-3xl">
          <Card className="p-8 text-center bg-zinc-900/50 border-red-800">
            <div className="w-16 h-16 rounded-full border-2 border-red-500 flex items-center justify-center mx-auto mb-4">
              <AlertCircle className="w-8 h-8 text-red-500" />
            </div>
            <h1 className="text-2xl font-bold mb-2">Booking Not Found</h1>
            <p className="text-muted-foreground mb-6">{error}</p>
            <Link href="/my-bookings">
              <Button>
                <ArrowLeft className="w-4 h-4 mr-2" />
                Back to My Bookings
              </Button>
            </Link>
          </Card>
        </div>
      </main>
    )
  }

  const { booking, event, show, zone } = bookingDetail
  const statusConfig = getStatusConfig(booking.status)
  const StatusIcon = statusConfig.icon
  const reference = generateReference(booking.id)
  const reservedAt = formatDateTime(booking.reserved_at)
  const confirmedAt = booking.confirmed_at ? formatDateTime(booking.confirmed_at) : null
  const expiresAt = formatDateTime(booking.expires_at)

  const isConfirmed = booking.status === "confirmed" || booking.status === "completed"
  const isPending = booking.status === "reserved" || booking.status === "pending"
  const isCancelled = booking.status === "cancelled" || booking.status === "expired"

  return (
    <main className="min-h-screen bg-background">
      <Header />

      <div className="container mx-auto px-4 lg:px-8 pt-24 pb-16 max-w-3xl">
        {/* Back Button */}
        <Link
          href="/my-bookings"
          className="inline-flex items-center gap-2 text-muted-foreground hover:text-foreground mb-6 transition-colors"
        >
          <ArrowLeft className="w-4 h-4" />
          Back to My Bookings
        </Link>

        <div className="space-y-6">
          {/* Header Section */}
          <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
            <div>
              <p className="text-sm text-muted-foreground mb-1">Booking Reference</p>
              <h1 className="text-2xl font-bold font-mono text-primary">{reference}</h1>
            </div>
            <Badge className={`${statusConfig.color} border text-sm px-4 py-2`}>
              <StatusIcon className="w-4 h-4 mr-2" />
              {statusConfig.label}
            </Badge>
          </div>

          {/* Status Description */}
          <Card className="p-4 bg-zinc-900/50 border-border/50">
            <p className="text-muted-foreground">{statusConfig.description}</p>
          </Card>

          {/* E-Ticket Section (only for confirmed bookings) */}
          {isConfirmed && (
            <Card className="p-6 bg-zinc-900/50 border-primary/30 overflow-hidden">
              <div className="flex flex-col items-center space-y-6">
                <div className="flex items-center gap-2 text-primary">
                  <QrCode className="w-5 h-5" />
                  <span className="font-semibold">E-Ticket</span>
                </div>

                {/* QR Code */}
                <div className="w-48 h-48 bg-white rounded-lg p-3 shadow-lg shadow-primary/20">
                  <svg viewBox="0 0 100 100" className="w-full h-full" xmlns="http://www.w3.org/2000/svg">
                    <rect width="100" height="100" fill="white" />
                    <g fill="black">
                      <rect x="10" y="10" width="25" height="25" />
                      <rect x="65" y="10" width="25" height="25" />
                      <rect x="10" y="65" width="25" height="25" />
                      <rect x="15" y="15" width="15" height="15" fill="white" />
                      <rect x="70" y="15" width="15" height="15" fill="white" />
                      <rect x="15" y="70" width="15" height="15" fill="white" />
                      <rect x="20" y="20" width="5" height="5" />
                      <rect x="75" y="20" width="5" height="5" />
                      <rect x="20" y="75" width="5" height="5" />
                      <rect x="45" y="10" width="5" height="5" />
                      <rect x="50" y="15" width="5" height="5" />
                      <rect x="45" y="25" width="5" height="5" />
                      <rect x="55" y="20" width="5" height="5" />
                      <rect x="40" y="40" width="20" height="20" />
                      <rect x="45" y="45" width="10" height="10" fill="white" />
                      <rect x="65" y="45" width="5" height="5" />
                      <rect x="75" y="50" width="5" height="5" />
                      <rect x="80" y="45" width="5" height="5" />
                      <rect x="70" y="60" width="5" height="5" />
                      <rect x="45" y="65" width="5" height="5" />
                      <rect x="55" y="70" width="5" height="5" />
                      <rect x="50" y="80" width="5" height="5" />
                      <rect x="65" y="75" width="5" height="5" />
                      <rect x="75" y="80" width="5" height="5" />
                      <rect x="80" y="70" width="5" height="5" />
                    </g>
                  </svg>
                </div>

                {/* Barcode */}
                <div className="w-full max-w-xs">
                  <svg
                    width="100%"
                    height="50"
                    viewBox="0 0 200 50"
                    xmlns="http://www.w3.org/2000/svg"
                    preserveAspectRatio="xMidYMid meet"
                  >
                    <rect width="200" height="50" fill="white" rx="4" />
                    <g fill="black">
                      {[...Array(40)].map((_, i) => (
                        <rect
                          key={i}
                          x={10 + i * 4.5}
                          y="8"
                          width={Math.random() > 0.5 ? 3 : 2}
                          height="34"
                        />
                      ))}
                    </g>
                  </svg>
                  <p className="text-xs text-center text-muted-foreground font-mono mt-1">{reference}</p>
                </div>

                <p className="text-sm text-muted-foreground">Scan at venue entrance</p>
              </div>
            </Card>
          )}

          {/* Event Info */}
          <Card className="overflow-hidden bg-zinc-900/50 border-border/50">
            <div className="flex flex-col sm:flex-row">
              {/* Event Image */}
              <div className="sm:w-48 h-48 sm:h-auto shrink-0">
                <img
                  src={event?.poster_url || "/placeholder.svg"}
                  alt={event?.name || "Event"}
                  className="w-full h-full object-cover"
                />
              </div>

              {/* Event Details */}
              <div className="flex-1 p-6 space-y-4">
                <div>
                  <h2 className="text-xl font-bold">{event?.name || "Event"}</h2>
                  {event?.slug && (
                    <Link
                      href={`/events/${event.slug}`}
                      className="text-sm text-primary hover:underline inline-flex items-center gap-1 mt-1"
                    >
                      View Event
                      <ExternalLink className="w-3 h-3" />
                    </Link>
                  )}
                </div>

                <div className="grid gap-3">
                  <div className="flex items-center gap-3 text-sm">
                    <Calendar className="w-4 h-4 text-primary shrink-0" />
                    <span>
                      {show?.show_date
                        ? new Date(show.show_date).toLocaleDateString("en-US", {
                            weekday: "long",
                            month: "long",
                            day: "numeric",
                            year: "numeric",
                          })
                        : "Date TBA"}
                    </span>
                  </div>
                  <div className="flex items-center gap-3 text-sm">
                    <Clock className="w-4 h-4 text-primary shrink-0" />
                    <span>
                      {show?.start_time
                        ? new Date(show.start_time).toLocaleTimeString("en-US", {
                            hour: "numeric",
                            minute: "2-digit",
                            hour12: true,
                          })
                        : "Time TBA"}
                    </span>
                  </div>
                  <div className="flex items-center gap-3 text-sm">
                    <MapPin className="w-4 h-4 text-primary shrink-0" />
                    <span>{event ? `${event.venue_name}, ${event.city}` : "Venue TBA"}</span>
                  </div>
                </div>
              </div>
            </div>
          </Card>

          {/* Ticket Info */}
          <Card className="p-6 bg-zinc-900/50 border-border/50">
            <h3 className="font-semibold mb-4 flex items-center gap-2">
              <Ticket className="w-5 h-5 text-primary" />
              Ticket Details
            </h3>
            <div className="space-y-3">
              <div className="flex justify-between">
                <span className="text-muted-foreground">Zone</span>
                <span className="font-medium">{zone?.name || "Standard"}</span>
              </div>
              <div className="flex justify-between">
                <span className="text-muted-foreground">Quantity</span>
                <span className="font-medium">{booking.quantity} ticket(s)</span>
              </div>
              <div className="flex justify-between">
                <span className="text-muted-foreground">Price per ticket</span>
                <span className="font-medium">฿{zone?.price?.toLocaleString() || (booking.total_price / booking.quantity).toLocaleString()}</span>
              </div>
              <div className="border-t border-border/50 pt-3 flex justify-between">
                <span className="font-semibold">Total</span>
                <span className="font-bold text-xl text-primary">฿{booking.total_price.toLocaleString()}</span>
              </div>
            </div>
          </Card>

          {/* Timeline */}
          <Card className="p-6 bg-zinc-900/50 border-border/50">
            <h3 className="font-semibold mb-4 flex items-center gap-2">
              <History className="w-5 h-5 text-primary" />
              Booking Timeline
            </h3>
            <div className="space-y-4">
              <div className="flex items-start gap-4">
                <div className="w-2 h-2 rounded-full bg-primary mt-2 shrink-0" />
                <div>
                  <p className="font-medium">Reserved</p>
                  <p className="text-sm text-muted-foreground">{reservedAt.date} at {reservedAt.time}</p>
                </div>
              </div>

              {confirmedAt && (
                <div className="flex items-start gap-4">
                  <div className="w-2 h-2 rounded-full bg-green-500 mt-2 shrink-0" />
                  <div>
                    <p className="font-medium">Confirmed</p>
                    <p className="text-sm text-muted-foreground">{confirmedAt.date} at {confirmedAt.time}</p>
                  </div>
                </div>
              )}

              {isPending && (
                <div className="flex items-start gap-4">
                  <div className="w-2 h-2 rounded-full bg-amber-500 mt-2 shrink-0" />
                  <div>
                    <p className="font-medium">Payment Required</p>
                    <p className="text-sm text-muted-foreground">Expires: {expiresAt.date} at {expiresAt.time}</p>
                  </div>
                </div>
              )}

              {isCancelled && (
                <div className="flex items-start gap-4">
                  <div className="w-2 h-2 rounded-full bg-red-500 mt-2 shrink-0" />
                  <div>
                    <p className="font-medium">{booking.status === "expired" ? "Expired" : "Cancelled"}</p>
                    <p className="text-sm text-muted-foreground">{expiresAt.date} at {expiresAt.time}</p>
                  </div>
                </div>
              )}
            </div>
          </Card>

          {/* Actions */}
          <div className="flex flex-col sm:flex-row gap-4">
            {isConfirmed && (
              <>
                <Button className="flex-1 bg-primary hover:bg-primary/90 text-primary-foreground">
                  <Download className="w-4 h-4 mr-2" />
                  Download E-Ticket
                </Button>
                <Button variant="outline" className="flex-1">
                  <Wallet className="w-4 h-4 mr-2" />
                  Add to Wallet
                </Button>
              </>
            )}

            {isPending && (
              <>
                <Link href={`/checkout?booking_id=${booking.id}`} className="flex-1">
                  <Button className="w-full bg-primary hover:bg-primary/90 text-primary-foreground">
                    <CreditCard className="w-4 h-4 mr-2" />
                    Complete Payment
                  </Button>
                </Link>
                <Button variant="outline" className="flex-1 border-red-500/50 text-red-400 hover:bg-red-500/10">
                  <XCircle className="w-4 h-4 mr-2" />
                  Cancel Booking
                </Button>
              </>
            )}

            {isCancelled && event?.slug && (
              <Link href={`/events/${event.slug}`} className="flex-1">
                <Button className="w-full bg-primary hover:bg-primary/90 text-primary-foreground">
                  <Ticket className="w-4 h-4 mr-2" />
                  Book Again
                </Button>
              </Link>
            )}
          </div>
        </div>
      </div>
    </main>
  )
}
