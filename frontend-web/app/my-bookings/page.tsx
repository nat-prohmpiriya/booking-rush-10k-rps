"use client"

import { useState, useEffect } from "react"
import { useRouter } from "next/navigation"
import Link from "next/link"
import { Header } from "@/components/header"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import { Skeleton } from "@/components/ui/skeleton"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import { useAuth } from "@/contexts/auth-context"
import { bookingApi } from "@/lib/api/booking"
import { eventsApi, zonesApi } from "@/lib/api/events"
import type { BookingResponse } from "@/lib/api/types"
import {
  Ticket,
  Calendar,
  Clock,
  MapPin,
  ChevronRight,
  Filter,
  Search,
  AlertCircle,
  CheckCircle2,
  XCircle,
  Timer,
} from "lucide-react"
import { Input } from "@/components/ui/input"

// Extended booking type with event and zone details
interface BookingWithEvent extends BookingResponse {
  event?: {
    id: string
    title: string
    venue: string
    date: string
    time: string
    image: string
    slug: string
  }
  zone?: {
    name: string
    price: number
  }
}

function BookingCardSkeleton() {
  return (
    <div className="glass rounded-xl p-4 sm:p-6 border border-border/50">
      <div className="flex flex-col sm:flex-row gap-4 sm:gap-6">
        <Skeleton className="h-32 sm:h-40 sm:w-56 rounded-lg shrink-0" />
        <div className="flex-1 space-y-4">
          <div className="space-y-2">
            <Skeleton className="h-6 w-3/4" />
            <Skeleton className="h-4 w-1/2" />
          </div>
          <div className="space-y-2">
            <Skeleton className="h-4 w-2/3" />
            <Skeleton className="h-4 w-1/2" />
          </div>
          <div className="flex gap-2">
            <Skeleton className="h-6 w-20" />
            <Skeleton className="h-6 w-24" />
          </div>
        </div>
        <div className="flex flex-row sm:flex-col justify-between sm:justify-center items-end gap-4">
          <Skeleton className="h-8 w-28" />
          <Skeleton className="h-10 w-32" />
        </div>
      </div>
    </div>
  )
}

function getStatusConfig(status: string) {
  switch (status) {
    case "confirmed":
      return {
        label: "Confirmed",
        color: "bg-green-500/20 text-green-400 border-green-500/30",
        icon: CheckCircle2,
      }
    case "pending":
      return {
        label: "Pending Payment",
        color: "bg-amber-500/20 text-amber-400 border-amber-500/30",
        icon: Timer,
      }
    case "completed":
      return {
        label: "Completed",
        color: "bg-blue-500/20 text-blue-400 border-blue-500/30",
        icon: CheckCircle2,
      }
    case "cancelled":
      return {
        label: "Cancelled",
        color: "bg-red-500/20 text-red-400 border-red-500/30",
        icon: XCircle,
      }
    default:
      return {
        label: status,
        color: "bg-gray-500/20 text-gray-400 border-gray-500/30",
        icon: AlertCircle,
      }
  }
}

function BookingCard({ booking }: { booking: BookingWithEvent }) {
  const statusConfig = getStatusConfig(booking.status)
  const StatusIcon = statusConfig.icon

  return (
    <Link href={`/my-bookings/${booking.id}`} className="block">
      <div className="group glass rounded-xl p-4 sm:p-6 border border-border/50 hover:border-primary/50 transition-all duration-300 cursor-pointer">
        <div className="flex flex-col sm:flex-row gap-4 sm:gap-6">
          {/* Event Image */}
          <div className="relative h-40 sm:h-40 sm:w-56 shrink-0 overflow-hidden rounded-lg">
            <img
              src={booking.event?.image || "/placeholder.svg"}
              alt={booking.event?.title || "Event"}
              className="w-full h-full object-cover transition-transform duration-500 group-hover:scale-110"
            />
            <div className="absolute top-2 right-2">
              <Badge className={`${statusConfig.color} border`}>
                <StatusIcon className="h-3 w-3 mr-1" />
                {statusConfig.label}
              </Badge>
            </div>
          </div>

          {/* Booking Info */}
          <div className="flex-1 space-y-3">
            <div>
              <h3 className="text-xl font-bold text-foreground group-hover:text-primary transition-colors line-clamp-1">
                {booking.event?.title || "Unknown Event"}
              </h3>
              <div className="flex items-center gap-2 text-muted-foreground text-sm mt-1">
                <MapPin className="h-4 w-4" />
                <span className="line-clamp-1">{booking.event?.venue || "TBA"}</span>
              </div>
            </div>

            <div className="flex flex-wrap gap-4 text-sm">
              <div className="flex items-center gap-2 text-muted-foreground">
                <Calendar className="h-4 w-4 text-primary" />
                <span>{booking.event?.date || "TBA"}</span>
              </div>
              <div className="flex items-center gap-2 text-muted-foreground">
                <Clock className="h-4 w-4 text-primary" />
                <span>{booking.event?.time || "TBA"}</span>
              </div>
            </div>

            <div className="flex flex-wrap items-center gap-3">
              <div className="flex items-center gap-2 text-sm">
                <Ticket className="h-4 w-4 text-primary" />
                <span className="text-foreground font-medium">{booking.zone?.name || "Standard"}</span>
                <span className="text-muted-foreground">× {booking.quantity}</span>
              </div>
            </div>

            <div className="text-xs text-muted-foreground">
              Booked on {new Date(booking.reserved_at).toLocaleDateString("en-US", {
                year: "numeric",
                month: "short",
                day: "numeric"
              })}
            </div>
          </div>

          {/* Price and Arrow */}
          <div className="flex flex-row sm:flex-col justify-between sm:justify-center items-end sm:items-end gap-4 pt-4 sm:pt-0 border-t sm:border-t-0 border-border/50">
            <div className="text-right">
              <p className="text-xs text-muted-foreground">Total</p>
              <p className="text-2xl font-bold bg-linear-to-r from-primary to-amber-400 bg-clip-text text-transparent">
                ฿{booking.total_price.toLocaleString()}
              </p>
            </div>

            <div className="flex items-center gap-2 text-muted-foreground group-hover:text-primary transition-colors">
              <span className="text-sm">View Details</span>
              <ChevronRight className="h-5 w-5" />
            </div>
          </div>
        </div>
      </div>
    </Link>
  )
}

type StatusFilter = "all" | "confirmed" | "pending" | "completed" | "cancelled"

export default function MyBookingsPage() {
  const router = useRouter()
  const { isAuthenticated, isLoading: authLoading } = useAuth()
  const [bookings, setBookings] = useState<BookingWithEvent[]>([])
  const [isLoading, setIsLoading] = useState(true)
  const [statusFilter, setStatusFilter] = useState<StatusFilter>("all")
  const [searchQuery, setSearchQuery] = useState("")

  useEffect(() => {
    if (!authLoading && !isAuthenticated) {
      router.push("/login?redirect=/my-bookings")
      return
    }

    async function fetchBookings() {
      setIsLoading(true)
      try {
        // Fetch bookings from API
        const apiBookings = await bookingApi.listUserBookings()

        if (!apiBookings || apiBookings.length === 0) {
          setBookings([])
          return
        }

        // Fetch event and zone details for each booking
        const bookingsWithDetails: BookingWithEvent[] = await Promise.all(
          apiBookings.map(async (booking) => {
            const bookingWithEvent: BookingWithEvent = { ...booking }

            // Fetch event details
            try {
              const event = await eventsApi.getById(booking.event_id)
              bookingWithEvent.event = {
                id: event.id,
                title: event.name,
                venue: `${event.venue_name}, ${event.city}`,
                date: new Date(event.created_at).toLocaleDateString("en-US", {
                  month: "short",
                  day: "numeric",
                  year: "numeric",
                }),
                time: "TBA", // Shows have specific times
                image: event.poster_url || "/placeholder.svg",
                slug: event.slug,
              }
            } catch (err) {
              console.warn(`Failed to fetch event ${booking.event_id}:`, err)
            }

            // Fetch zone details
            try {
              const zone = await zonesApi.getById(booking.zone_id)
              bookingWithEvent.zone = {
                name: zone.name,
                price: zone.price,
              }
            } catch (err) {
              console.warn(`Failed to fetch zone ${booking.zone_id}:`, err)
            }

            return bookingWithEvent
          })
        )

        setBookings(bookingsWithDetails)
      } catch (error) {
        console.error("Failed to fetch bookings:", error)
        setBookings([])
      } finally {
        setIsLoading(false)
      }
    }

    if (isAuthenticated) {
      fetchBookings()
    }
  }, [isAuthenticated, authLoading, router])

  // Filter bookings
  const filteredBookings = bookings.filter((booking) => {
    const matchesStatus = statusFilter === "all" || booking.status === statusFilter
    const matchesSearch =
      searchQuery === "" ||
      booking.event?.title?.toLowerCase().includes(searchQuery.toLowerCase()) ||
      booking.event?.venue?.toLowerCase().includes(searchQuery.toLowerCase())
    return matchesStatus && matchesSearch
  })

  // Group bookings by status for summary
  const bookingSummary = {
    total: bookings.length,
    confirmed: bookings.filter((b) => b.status === "confirmed").length,
    pending: bookings.filter((b) => b.status === "pending").length,
    completed: bookings.filter((b) => b.status === "completed").length,
    cancelled: bookings.filter((b) => b.status === "cancelled").length,
  }

  if (authLoading) {
    return (
      <main className="min-h-screen bg-background">
        <Header />
        <div className="container mx-auto px-4 lg:px-8 pt-24 pb-16">
          <div className="flex items-center justify-center h-64">
            <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary" />
          </div>
        </div>
      </main>
    )
  }

  return (
    <main className="min-h-screen bg-background">
      <Header />

      {/* Hero Section */}
      <section className="relative pt-24 pb-12 lg:pt-32 lg:pb-16 overflow-hidden">
        {/* Background */}
        <div className="absolute inset-0 bg-[radial-gradient(ellipse_at_top,var(--tw-gradient-stops))] from-primary/20 via-background to-background" />
        <div className="absolute inset-0 bg-[url('/images/grid-pattern.svg')] opacity-5" />

        <div className="container mx-auto px-4 lg:px-8 relative z-10">
          <div className="max-w-3xl mx-auto text-center space-y-6">
            <div className="inline-block glass px-4 py-2 rounded-full">
              <span className="text-primary text-sm font-medium flex items-center gap-2">
                <Ticket className="h-4 w-4" />
                My Bookings
              </span>
            </div>
            <h1 className="text-4xl lg:text-5xl font-bold text-balance">
              Your{" "}
              <span className="bg-linear-to-r from-primary to-amber-400 bg-clip-text text-transparent">
                Tickets & Bookings
              </span>
            </h1>
            <p className="text-lg text-muted-foreground max-w-xl mx-auto text-pretty">
              Manage all your event bookings, view tickets, and track your upcoming experiences.
            </p>
          </div>
        </div>
      </section>

      {/* Stats Summary */}
      <section className="container mx-auto px-4 lg:px-8 -mt-4 mb-8">
        <div className="grid grid-cols-2 sm:grid-cols-4 gap-4">
          <div className="glass rounded-xl p-4 border border-border/50 text-center">
            <p className="text-3xl font-bold bg-linear-to-r from-primary to-amber-400 bg-clip-text text-transparent">
              {bookingSummary.total}
            </p>
            <p className="text-sm text-muted-foreground">Total Bookings</p>
          </div>
          <div className="glass rounded-xl p-4 border border-green-500/30 text-center">
            <p className="text-3xl font-bold text-green-400">{bookingSummary.confirmed}</p>
            <p className="text-sm text-muted-foreground">Confirmed</p>
          </div>
          <div className="glass rounded-xl p-4 border border-amber-500/30 text-center">
            <p className="text-3xl font-bold text-amber-400">{bookingSummary.pending}</p>
            <p className="text-sm text-muted-foreground">Pending</p>
          </div>
          <div className="glass rounded-xl p-4 border border-blue-500/30 text-center">
            <p className="text-3xl font-bold text-blue-400">{bookingSummary.completed}</p>
            <p className="text-sm text-muted-foreground">Completed</p>
          </div>
        </div>
      </section>

      {/* Bookings Section */}
      <section className="container mx-auto px-4 lg:px-8 pb-16 lg:pb-24">
        {/* Filters */}
        <div className="flex flex-col sm:flex-row gap-4 mb-8">
          {/* Search */}
          <div className="relative flex-1">
            <Search className="absolute left-4 top-1/2 -translate-y-1/2 h-5 w-5 text-muted-foreground" />
            <Input
              type="text"
              placeholder="Search bookings..."
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              className="pl-12 glass border-primary/30 focus:border-primary"
            />
          </div>

          {/* Status Filter */}
          <Select value={statusFilter} onValueChange={(v) => setStatusFilter(v as StatusFilter)}>
            <SelectTrigger className="w-full sm:w-48 border-primary/30">
              <Filter className="h-4 w-4 mr-2" />
              <SelectValue placeholder="Filter by status" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">All Bookings</SelectItem>
              <SelectItem value="confirmed">Confirmed</SelectItem>
              <SelectItem value="pending">Pending</SelectItem>
              <SelectItem value="completed">Completed</SelectItem>
              <SelectItem value="cancelled">Cancelled</SelectItem>
            </SelectContent>
          </Select>
        </div>

        {/* Bookings List */}
        {isLoading ? (
          <div className="space-y-4">
            <BookingCardSkeleton />
            <BookingCardSkeleton />
            <BookingCardSkeleton />
          </div>
        ) : filteredBookings.length > 0 ? (
          <div className="space-y-4">
            {filteredBookings.map((booking) => (
              <BookingCard key={booking.id} booking={booking} />
            ))}
          </div>
        ) : (
          <div className="text-center py-16 space-y-4">
            <div className="glass inline-block p-6 rounded-full">
              <Ticket className="h-12 w-12 text-muted-foreground" />
            </div>
            <h3 className="text-2xl font-semibold text-foreground">No bookings found</h3>
            <p className="text-muted-foreground max-w-md mx-auto">
              {statusFilter !== "all"
                ? `You don't have any ${statusFilter} bookings.`
                : "You haven't made any bookings yet. Start exploring events!"}
            </p>
            <Link href="/events">
              <Button className="mt-4 bg-linear-to-r from-primary to-amber-400 text-primary-foreground">
                Browse Events
              </Button>
            </Link>
          </div>
        )}
      </section>
    </main>
  )
}
