"use client"

import { useState, useEffect, useCallback, Suspense } from "react"
import { Card } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Separator } from "@/components/ui/separator"
import { Clock, Shield, Lock, Calendar, MapPin, Ticket, AlertTriangle, CreditCard } from "lucide-react"
import { useRouter, useSearchParams } from "next/navigation"
import { bookingApi, paymentApi } from "@/lib/api/booking"
import { paymentApi as paymentMethodsApi, type PaymentMethod } from "@/lib/api/payment"
import { eventsApi, zonesApi } from "@/lib/api/events"
import type { EventResponse, ShowResponse, ShowZoneResponse, ReserveSeatsResponse, PaymentIntentResponse } from "@/lib/api/types"
import { ApiRequestError } from "@/lib/api/client"
import { getStripe, isStripeConfigured } from "@/lib/stripe"
import { StripePaymentForm } from "@/components/payment/stripe-payment-form"
import type { Stripe } from "@stripe/stripe-js"

type CheckoutState = "loading" | "reserving" | "creating_intent" | "ready" | "processing" | "success" | "error" | "timeout"

interface QueueData {
  eventId: string
  showId: string
  tickets: Record<string, number>
  total: number
  queuePass: string
  queuePassExpiresAt: string
}

// Wrapper component to handle Suspense for useSearchParams
export default function CheckoutPage() {
  return (
    <Suspense fallback={<CheckoutLoadingFallback />}>
      <CheckoutContent />
    </Suspense>
  )
}

function CheckoutLoadingFallback() {
  return (
    <div className="min-h-screen bg-black-gradient pattern-dots flex items-center justify-center">
      <div className="text-center">
        <div className="w-12 h-12 border-4 border-primary border-t-transparent rounded-full animate-spin mx-auto mb-4" />
        <p className="text-muted-foreground">Loading checkout...</p>
      </div>
    </div>
  )
}

function CheckoutContent() {
  const router = useRouter()
  const searchParams = useSearchParams()
  const bookingIdParam = searchParams.get("booking_id")

  // Mode: "queue" (normal flow) or "direct" (from booking detail)
  const [mode, setMode] = useState<"queue" | "direct" | null>(null)

  // Queue data from sessionStorage
  const [queueData, setQueueData] = useState<QueueData | null>(null)

  // Event/Show/Zone data
  const [event, setEvent] = useState<EventResponse | null>(null)
  const [show, setShow] = useState<ShowResponse | null>(null)
  const [zones, setZones] = useState<ShowZoneResponse[]>([])

  // Booking state
  const [checkoutState, setCheckoutState] = useState<CheckoutState>("loading")
  const [reservation, setReservation] = useState<ReserveSeatsResponse | null>(null)
  const [error, setError] = useState<string>("")

  // Stripe state
  const [stripe, setStripe] = useState<Stripe | null>(null)
  const [paymentIntent, setPaymentIntent] = useState<PaymentIntentResponse | null>(null)

  // Saved payment methods
  const [savedPaymentMethods, setSavedPaymentMethods] = useState<PaymentMethod[]>([])
  const [selectedPaymentMethod, setSelectedPaymentMethod] = useState<string | null>(null) // null = new card

  // Timer state
  const [timeLeft, setTimeLeft] = useState(600) // 10 minutes default

  // Initialize Stripe
  useEffect(() => {
    if (isStripeConfigured()) {
      getStripe().then(setStripe)
    }
  }, [])

  // Fetch saved payment methods
  useEffect(() => {
    const fetchSavedPaymentMethods = async () => {
      try {
        const response = await paymentMethodsApi.listPaymentMethods()
        setSavedPaymentMethods(response.payment_methods || [])
        // Auto-select default payment method if available
        const defaultMethod = response.payment_methods?.find(pm => pm.is_default)
        if (defaultMethod) {
          setSelectedPaymentMethod(defaultMethod.id)
        }
      } catch (err) {
        // Silently fail - user can still use new card
        console.log("No saved payment methods:", err)
      }
    }

    fetchSavedPaymentMethods()
  }, [])

  // Load data - either from booking_id param (direct) or sessionStorage (queue)
  useEffect(() => {
    if (typeof window === "undefined") return

    // Direct mode: Load existing booking
    if (bookingIdParam) {
      setMode("direct")
      const loadExistingBooking = async () => {
        try {
          const booking = await bookingApi.getBooking(bookingIdParam)

          // Check if booking is still pending/reserved
          if (booking.status !== "reserved" && booking.status !== "pending") {
            setError(`This booking is ${booking.status}. Cannot complete payment.`)
            setCheckoutState("error")
            return
          }

          // Load event details
          const eventData = await eventsApi.getEvent(booking.event_id)
          setEvent(eventData)

          // Load zone details
          try {
            const zoneData = await zonesApi.getById(booking.zone_id)
            setZones([zoneData])
          } catch {
            console.warn("Could not load zone details")
          }

          // Set reservation from existing booking
          setReservation({
            booking_id: booking.id,
            status: booking.status,
            expires_at: booking.expires_at,
            total_price: booking.total_price,
          })

          // Calculate time left from booking expiry
          if (booking.expires_at) {
            const expiresAt = new Date(booking.expires_at).getTime()
            const now = Date.now()
            const remaining = Math.max(0, Math.floor((expiresAt - now) / 1000))
            setTimeLeft(remaining)

            if (remaining <= 0) {
              setError("This booking has expired.")
              setCheckoutState("error")
              return
            }
          }

          // Skip straight to creating payment intent
          setCheckoutState("creating_intent")
        } catch (err) {
          console.error("Failed to load booking:", err)
          setError("Failed to load booking details. Please try again.")
          setCheckoutState("error")
        }
      }

      loadExistingBooking()
      return
    }

    // Queue mode: Load from sessionStorage
    setMode("queue")
    const eventId = sessionStorage.getItem("queue_event_id")
    const showId = sessionStorage.getItem("queue_show_id")
    const ticketsStr = sessionStorage.getItem("queue_tickets")
    const totalStr = sessionStorage.getItem("queue_total")
    const queuePass = sessionStorage.getItem("queue_pass")
    const queuePassExpiresAt = sessionStorage.getItem("queue_pass_expires_at")

    if (!eventId || !queuePass) {
      setError("No queue pass found. Please join the queue first.")
      setCheckoutState("error")
      return
    }

    const tickets = ticketsStr ? JSON.parse(ticketsStr) : {}
    const total = totalStr ? parseInt(totalStr, 10) : 0

    setQueueData({
      eventId,
      showId: showId || "",
      tickets,
      total,
      queuePass,
      queuePassExpiresAt: queuePassExpiresAt || "",
    })

    // Calculate time left from queue pass expiry
    if (queuePassExpiresAt) {
      const expiresAt = new Date(queuePassExpiresAt).getTime()
      const now = Date.now()
      const remaining = Math.max(0, Math.floor((expiresAt - now) / 1000))
      setTimeLeft(remaining)
    }
  }, [bookingIdParam])

  // Fetch event details
  useEffect(() => {
    if (!queueData?.eventId) return

    const fetchEventDetails = async () => {
      try {
        const eventData = await eventsApi.getEvent(queueData.eventId)
        setEvent(eventData)

        if (queueData.showId && eventData.slug) {
          // Use slug from fetched event to get shows
          const shows = await eventsApi.getEventShowsBySlug(eventData.slug)
          const showData = shows.find((s: ShowResponse) => s.id === queueData.showId)
          if (showData) {
            setShow(showData)
            const zonesData = await eventsApi.getShowZones(showData.id)
            setZones(zonesData)
          }
        }

        setCheckoutState("reserving")
      } catch (err) {
        console.error("Failed to fetch event details:", err)
        setError("Failed to load event details")
        setCheckoutState("error")
      }
    }

    fetchEventDetails()
  }, [queueData])

  // Reserve seats when ready
  useEffect(() => {
    if (checkoutState !== "reserving" || !queueData || !event) return

    const reserveSeats = async () => {
      try {
        // Get first zone from tickets (simplified - in real app may have multiple zones)
        const zoneEntries = Object.entries(queueData.tickets)
        if (zoneEntries.length === 0) {
          setError("No tickets selected")
          setCheckoutState("error")
          return
        }

        const [zoneId, quantity] = zoneEntries[0]
        const zone = zones.find(z => z.id === zoneId)

        const reservationData = await bookingApi.reserveSeats(
          {
            event_id: queueData.eventId,
            zone_id: zoneId,
            show_id: queueData.showId || undefined,
            quantity: quantity as number,
            unit_price: zone?.price,
          },
          queueData.queuePass
        )

        setReservation(reservationData)

        // Update timer based on reservation expiry
        if (reservationData.expires_at) {
          const expiresAt = new Date(reservationData.expires_at).getTime()
          const now = Date.now()
          const remaining = Math.max(0, Math.floor((expiresAt - now) / 1000))
          setTimeLeft(remaining)
        }

        setCheckoutState("creating_intent")
      } catch (err) {
        console.error("Failed to reserve seats:", err)
        if (err instanceof ApiRequestError) {
          if (err.code === "QUEUE_PASS_EXPIRED" || err.code === "INVALID_QUEUE_PASS") {
            setError("Your queue pass has expired. Please rejoin the queue.")
          } else if (err.code === "SEATS_NOT_AVAILABLE") {
            setError("Sorry, the requested seats are no longer available.")
          } else {
            setError(err.message)
          }
        } else {
          setError("Failed to reserve seats")
        }
        setCheckoutState("error")
      }
    }

    reserveSeats()
  }, [checkoutState, queueData, event, zones])

  // Create PaymentIntent after reservation
  useEffect(() => {
    if (checkoutState !== "creating_intent" || !reservation) return

    const createPaymentIntent = async () => {
      try {
        const orderSummary = getOrderSummary()

        const intentData = await paymentApi.createPaymentIntent({
          booking_id: reservation.booking_id,
          amount: orderSummary.total,
          currency: "THB",
        })

        setPaymentIntent(intentData)
        setCheckoutState("ready")
      } catch (err) {
        console.error("Failed to create payment intent:", err)
        if (err instanceof ApiRequestError) {
          setError(err.message)
        } else {
          setError("Failed to initialize payment. Please try again.")
        }
        setCheckoutState("error")
      }
    }

    createPaymentIntent()
  }, [checkoutState, reservation])

  // Countdown timer
  useEffect(() => {
    if (checkoutState !== "ready" || timeLeft <= 0) return

    const timer = setInterval(() => {
      setTimeLeft((prev) => {
        if (prev <= 1) {
          clearInterval(timer)
          setCheckoutState("timeout")
          return 0
        }
        return prev - 1
      })
    }, 1000)

    return () => clearInterval(timer)
  }, [checkoutState, timeLeft])

  // Handle timeout - release reservation and redirect
  useEffect(() => {
    if (checkoutState !== "timeout") return

    const handleTimeout = async () => {
      if (reservation?.booking_id) {
        try {
          await bookingApi.releaseBooking(reservation.booking_id)
        } catch (err) {
          console.error("Failed to release booking:", err)
        }
      }

      // Clear session storage
      clearQueueSession()

      // Redirect after short delay
      setTimeout(() => {
        router.push("/")
      }, 3000)
    }

    handleTimeout()
  }, [checkoutState, reservation, router])

  const clearQueueSession = () => {
    if (typeof window === "undefined") return
    sessionStorage.removeItem("queue_token")
    sessionStorage.removeItem("queue_event_id")
    sessionStorage.removeItem("queue_show_id")
    sessionStorage.removeItem("queue_tickets")
    sessionStorage.removeItem("queue_total")
    sessionStorage.removeItem("queue_pass")
    sessionStorage.removeItem("queue_pass_expires_at")
  }

  const formatTime = (seconds: number) => {
    const mins = Math.floor(seconds / 60)
    const secs = seconds % 60
    return `${mins.toString().padStart(2, "0")}:${secs.toString().padStart(2, "0")}`
  }

  const isUrgent = timeLeft < 120 // Less than 2 minutes

  // Calculate totals
  const getOrderSummary = useCallback(() => {
    // Direct mode: use reservation data
    if (mode === "direct" && reservation) {
      const subtotal = reservation.total_price
      const serviceFee = 0 // No service fee - platform absorbs payment processing costs
      const total = subtotal

      // Try to get zone info for display
      const zone = zones.length > 0 ? zones[0] : null
      const items = zone ? [{
        zoneId: zone.id,
        zoneName: zone.name || "Zone",
        quantity: Math.round(subtotal / (zone.price || subtotal)), // Calculate quantity from price
        price: zone.price || subtotal,
        subtotal: subtotal,
      }] : [{
        zoneId: "unknown",
        zoneName: "Ticket",
        quantity: 1,
        price: subtotal,
        subtotal: subtotal,
      }]

      return { items, subtotal, serviceFee, total }
    }

    // Queue mode: calculate from ticket selection
    if (!queueData || !zones.length) {
      return { items: [], subtotal: 0, serviceFee: 0, total: 0 }
    }

    const items = Object.entries(queueData.tickets).map(([zoneId, quantity]) => {
      const zone = zones.find(z => z.id === zoneId)
      return {
        zoneId,
        zoneName: zone?.name || "Unknown Zone",
        quantity: quantity as number,
        price: zone?.price || 0,
        subtotal: (zone?.price || 0) * (quantity as number),
      }
    })

    const subtotal = items.reduce((sum, item) => sum + item.subtotal, 0)
    const serviceFee = 0 // No service fee - platform absorbs payment processing costs
    const total = subtotal

    return { items, subtotal, serviceFee, total }
  }, [mode, reservation, queueData, zones])

  const orderSummary = getOrderSummary()

  // Handle successful Stripe payment
  const handlePaymentSuccess = async (paymentIntentId: string) => {
    // Guard against duplicate calls
    if (checkoutState === "processing" || checkoutState === "success") return
    if (!reservation?.booking_id || !paymentIntent?.payment_id) return

    setCheckoutState("processing")

    try {
      // Confirm payment on backend
      await paymentApi.confirmPaymentIntent({
        payment_id: paymentIntent.payment_id,
        payment_intent_id: paymentIntentId,
      })

      // Confirm booking
      await bookingApi.confirmBooking(reservation.booking_id, {
        payment_id: paymentIntent.payment_id,
      })

      setCheckoutState("success")

      // Clear session and redirect to confirmation
      clearQueueSession()

      setTimeout(() => {
        router.push(`/booking/confirmation?booking_id=${reservation.booking_id}`)
      }, 2000)
    } catch (err) {
      console.error("Failed to confirm payment:", err)
      if (err instanceof ApiRequestError) {
        setError(err.message)
      } else {
        setError("Payment completed but confirmation failed. Please contact support.")
      }
      setCheckoutState("error")
    }
  }

  // Handle payment error
  const handlePaymentError = (errorMessage: string) => {
    setError(errorMessage)
  }

  // Handle cancel
  const handleCancel = async () => {
    if (reservation?.booking_id) {
      try {
        await bookingApi.releaseBooking(reservation.booking_id)
      } catch (err) {
        console.error("Failed to release booking:", err)
      }
    }

    clearQueueSession()
    router.push("/")
  }

  // Loading state
  if (checkoutState === "loading" || checkoutState === "reserving" || checkoutState === "creating_intent") {
    return (
      <div className="min-h-screen bg-[#0a0a0a] flex items-center justify-center">
        <div className="text-center space-y-4">
          <div className="w-16 h-16 border-4 border-[#d4af37]/30 border-t-[#d4af37] rounded-full animate-spin mx-auto" />
          <p className="text-gray-400">
            {checkoutState === "loading" && "Loading checkout..."}
            {checkoutState === "reserving" && "Reserving your seats..."}
            {checkoutState === "creating_intent" && "Initializing payment..."}
          </p>
        </div>
      </div>
    )
  }

  // Error state
  if (checkoutState === "error") {
    const isMaxTicketsError = error?.toLowerCase().includes("maximum tickets")

    return (
      <div className="min-h-screen bg-[#0a0a0a] flex items-center justify-center p-4">
        {/* Background decorations */}
        <div className="absolute inset-0 overflow-hidden pointer-events-none">
          <div className="absolute top-1/4 left-1/4 w-96 h-96 bg-red-500/5 rounded-full blur-3xl" />
          <div className="absolute bottom-1/4 right-1/4 w-96 h-96 bg-orange-500/5 rounded-full blur-3xl" />
        </div>

        <div className="relative z-10 w-full max-w-lg">
          {/* Main Card */}
          <Card className="bg-gradient-to-b from-[#1a1a1a] to-[#141414] border border-red-900/30 overflow-hidden">
            {/* Top accent line */}
            <div className="h-1 bg-gradient-to-r from-red-600 via-orange-500 to-red-600" />

            <div className="p-8 md:p-10">
              {/* Icon */}
              <div className="relative mx-auto mb-6 w-20 h-20">
                <div className="absolute inset-0 bg-red-500/30 rounded-full blur-xl animate-pulse" />
                <div className="relative w-20 h-20 rounded-full bg-gradient-to-br from-red-700 to-red-800 border border-red-400/50 flex items-center justify-center">
                  <AlertTriangle className="w-9 h-9 text-white" />
                </div>
              </div>

              {/* Title */}
              <h1 className="text-2xl md:text-3xl font-bold text-white text-center mb-3">
                Checkout Failed
              </h1>

              {/* Error message */}
              <div className="bg-red-950/30 border border-red-900/30 rounded-xl p-4 mb-6">
                <p className="text-red-300 text-center text-sm md:text-base">
                  {error}
                </p>
              </div>

              {/* Helpful tips based on error type */}
              {isMaxTicketsError && (
                <div className="bg-[#1f1f1f] rounded-xl p-4 mb-6 border border-gray-800">
                  <p className="text-gray-400 text-sm mb-3 flex items-center gap-2">
                    <span className="text-[#d4af37]">💡</span>
                    <span className="font-medium text-gray-300">What can you do?</span>
                  </p>
                  <ul className="text-sm text-gray-400 space-y-2">
                    <li className="flex items-start gap-2">
                      <span className="text-gray-600 mt-0.5">•</span>
                      <span>Check your existing bookings in your profile</span>
                    </li>
                    <li className="flex items-start gap-2">
                      <span className="text-gray-600 mt-0.5">•</span>
                      <span>Cancel unused reservations to free up your limit</span>
                    </li>
                    <li className="flex items-start gap-2">
                      <span className="text-gray-600 mt-0.5">•</span>
                      <span>Try booking for a different event</span>
                    </li>
                  </ul>
                </div>
              )}

              {/* Action buttons */}
              <div className="space-y-3">
                <Button
                  onClick={() => router.push("/")}
                  className="w-full py-6 text-base font-semibold bg-gradient-to-r from-[#d4af37] to-[#c9a030] hover:from-[#e5c048] hover:to-[#d4af37] text-black shadow-lg shadow-[#d4af37]/20 transition-all duration-300"
                >
                  Browse Other Events
                </Button>

                {isMaxTicketsError && (
                  <Button
                    variant="outline"
                    onClick={() => router.push("/profile/bookings")}
                    className="w-full py-5 text-base border-gray-700 text-gray-300 hover:bg-gray-800 hover:text-white"
                  >
                    View My Bookings
                  </Button>
                )}

                <Button
                  variant="ghost"
                  onClick={() => router.back()}
                  className="w-full text-sm text-gray-500 hover:text-gray-300 py-2"
                >
                  ← Go back
                </Button>
              </div>
            </div>

            {/* Bottom decoration */}
            <div className="px-8 pb-6">
              <div className="border-t border-gray-800 pt-4">
                <p className="text-xs text-gray-600 text-center">
                  Need help?{" "}
                  <a href="/support" className="text-[#d4af37] hover:underline">
                    Contact Support
                  </a>
                </p>
              </div>
            </div>
          </Card>
        </div>
      </div>
    )
  }

  // Timeout state
  if (checkoutState === "timeout") {
    return (
      <div className="min-h-screen bg-[#0a0a0a] flex items-center justify-center p-4">
        <Card className="bg-[#1a1a1a] border-yellow-800 p-8 max-w-md text-center">
          <div className="w-16 h-16 rounded-full border-2 border-yellow-500 flex items-center justify-center mx-auto mb-4">
            <Clock className="w-8 h-8 text-yellow-500" />
          </div>
          <h1 className="text-2xl font-bold text-white mb-2">Session Expired</h1>
          <p className="text-gray-400 mb-6">
            Your reservation has timed out. Your seats have been released. Redirecting...
          </p>
        </Card>
      </div>
    )
  }

  // Success state
  if (checkoutState === "success") {
    return (
      <div className="min-h-screen bg-[#0a0a0a] flex items-center justify-center p-4">
        <Card className="bg-[#1a1a1a] border-green-800 p-8 max-w-md text-center">
          <div className="w-16 h-16 rounded-full border-2 border-green-500 flex items-center justify-center mx-auto mb-4">
            <svg className="w-8 h-8 text-green-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
            </svg>
          </div>
          <h1 className="text-2xl font-bold text-white mb-2">Payment Successful!</h1>
          <p className="text-gray-400 mb-6">Redirecting to your booking confirmation...</p>
        </Card>
      </div>
    )
  }

  // Processing state
  if (checkoutState === "processing") {
    return (
      <div className="min-h-screen bg-[#0a0a0a] flex items-center justify-center">
        <div className="text-center space-y-4">
          <div className="w-16 h-16 border-4 border-[#d4af37]/30 border-t-[#d4af37] rounded-full animate-spin mx-auto" />
          <p className="text-gray-400">Confirming your payment...</p>
        </div>
      </div>
    )
  }

  // Ready state - main checkout form
  return (
    <div className="min-h-screen bg-[#0a0a0a]">
      <div className="mx-auto max-w-7xl px-4 py-8 sm:px-6 lg:px-8">
        {/* Header */}
        <div className="mb-8">
          <h1 className="text-3xl font-bold text-white">Checkout</h1>
          <p className="mt-2 text-gray-400">Complete your booking securely</p>
        </div>

        {/* Two Column Layout */}
        <div className="grid gap-8 lg:grid-cols-2">
          {/* Left Column - Order Summary */}
          <div className="lg:order-1">
            <Card className="overflow-hidden border-0 bg-[#141414]">
              <div className="p-6">
                <h2 className="mb-6 text-xl font-semibold text-white">Order Summary</h2>

                {/* Event Image */}
                {event?.banner_url && (
                  <div className="mb-6 overflow-hidden rounded-lg">
                    <img src={event.banner_url} alt={event.name} className="h-48 w-full object-cover" />
                  </div>
                )}

                {/* Event Details */}
                <div className="space-y-4">
                  <div>
                    <h3 className="text-lg font-semibold text-white">{event?.name || "Event"}</h3>
                  </div>

                  <div className="flex items-start gap-3 text-sm text-gray-300">
                    <Calendar className="mt-0.5 h-4 w-4 shrink-0 text-[#d4af37]" />
                    <div>
                      <div>
                        {show?.show_date
                          ? new Date(show.show_date).toLocaleDateString("en-US", {
                              weekday: "long",
                              month: "long",
                              day: "numeric",
                              year: "numeric",
                            })
                          : "Date TBA"}
                      </div>
                      <div className="text-gray-400">
                        {show?.start_time
                          ? new Date(show.start_time).toLocaleTimeString("en-US", {
                              hour: "numeric",
                              minute: "2-digit",
                              hour12: true,
                            })
                          : "Time TBA"}
                      </div>
                    </div>
                  </div>

                  <div className="flex items-start gap-3 text-sm text-gray-300">
                    <MapPin className="mt-0.5 h-4 w-4 shrink-0 text-[#d4af37]" />
                    <div>{event?.venue_name || "Venue TBA"}</div>
                  </div>

                  <div className="flex items-start gap-3 text-sm text-gray-300">
                    <Ticket className="mt-0.5 h-4 w-4 shrink-0 text-[#d4af37]" />
                    <div>
                      {orderSummary.items.map((item) => (
                        <div key={item.zoneId}>
                          <div>{item.zoneName}</div>
                          <div className="text-gray-400">Quantity: {item.quantity}</div>
                        </div>
                      ))}
                    </div>
                  </div>
                </div>

                <Separator className="my-6 bg-gray-700" />

                {/* Price Breakdown */}
                <div className="space-y-3">
                  {orderSummary.items.map((item) => (
                    <div key={item.zoneId} className="flex justify-between text-sm text-gray-300">
                      <span>
                        {item.zoneName} x {item.quantity}
                      </span>
                      <span>฿{item.subtotal.toLocaleString()}</span>
                    </div>
                  ))}
                  <Separator className="bg-gray-700" />

                  <div className="flex justify-between text-lg font-bold text-white">
                    <span>Total</span>
                    <span className="text-[#d4af37]">฿{orderSummary.total.toLocaleString()}</span>
                  </div>
                </div>
              </div>
            </Card>
          </div>

          {/* Right Column - Payment Section */}
          <div className="lg:order-2">
            <Card className="border-0 bg-[#141414]">
              <div className="p-6">
                {/* Countdown Timer */}
                <div
                  className={`mb-6 flex items-center justify-center gap-2 rounded-lg p-4 ${
                    isUrgent ? "bg-red-950/50" : "bg-gray-800/50"
                  }`}
                >
                  <Clock className={`h-5 w-5 ${isUrgent ? "text-red-400" : "text-gray-400"}`} />
                  <span className={`font-mono text-lg font-semibold ${isUrgent ? "text-red-400" : "text-gray-300"}`}>
                    Complete in {formatTime(timeLeft)}
                  </span>
                </div>

                <h2 className="mb-6 text-xl font-semibold text-white">Payment Details</h2>

                {/* Saved Payment Methods */}
                {savedPaymentMethods.length > 0 && (
                  <div className="mb-6 space-y-3">
                    <p className="text-sm font-medium text-gray-300">Saved Cards</p>
                    {savedPaymentMethods.map((pm) => (
                      <Button
                        key={pm.id}
                        type="button"
                        variant="outline"
                        onClick={() => setSelectedPaymentMethod(pm.id)}
                        className={`w-full h-auto flex items-center gap-3 p-4 transition-colors ${
                          selectedPaymentMethod === pm.id
                            ? "border-[#d4af37] bg-[#d4af37]/10"
                            : "border-gray-700 bg-black/30 hover:border-gray-600"
                        }`}
                      >
                        <div className={`w-5 h-5 rounded-full border-2 flex items-center justify-center ${
                          selectedPaymentMethod === pm.id ? "border-[#d4af37]" : "border-gray-500"
                        }`}>
                          {selectedPaymentMethod === pm.id && (
                            <div className="w-2.5 h-2.5 rounded-full bg-[#d4af37]" />
                          )}
                        </div>
                        <CreditCard className="h-5 w-5 text-gray-400" />
                        <div className="flex-1 text-left">
                          <span className="text-white capitalize">{pm.brand}</span>
                          <span className="text-gray-400"> •••• {pm.last4}</span>
                          {pm.is_default && (
                            <span className="ml-2 text-xs bg-[#d4af37]/20 text-[#d4af37] px-2 py-0.5 rounded">
                              Default
                            </span>
                          )}
                        </div>
                        <span className="text-sm text-gray-500">
                          {pm.exp_month.toString().padStart(2, "0")}/{pm.exp_year.toString().slice(-2)}
                        </span>
                      </Button>
                    ))}

                    {/* Use new card option */}
                    <Button
                      type="button"
                      variant="outline"
                      onClick={() => setSelectedPaymentMethod(null)}
                      className={`w-full h-auto flex items-center gap-3 p-4 transition-colors ${
                        selectedPaymentMethod === null
                          ? "border-[#d4af37] bg-[#d4af37]/10"
                          : "border-gray-700 bg-black/30 hover:border-gray-600"
                      }`}
                    >
                      <div className={`w-5 h-5 rounded-full border-2 flex items-center justify-center ${
                        selectedPaymentMethod === null ? "border-[#d4af37]" : "border-gray-500"
                      }`}>
                        {selectedPaymentMethod === null && (
                          <div className="w-2.5 h-2.5 rounded-full bg-[#d4af37]" />
                        )}
                      </div>
                      <CreditCard className="h-5 w-5 text-gray-400" />
                      <span className="text-white">Use a new card</span>
                    </Button>
                  </div>
                )}

                {/* Stripe Payment Form or Fallback */}
                {stripe && paymentIntent?.client_secret ? (
                  <StripePaymentForm
                    stripe={stripe}
                    clientSecret={paymentIntent.client_secret}
                    amount={orderSummary.total}
                    onSuccess={handlePaymentSuccess}
                    onError={handlePaymentError}
                    disabled={timeLeft <= 0}
                    savedPaymentMethods={savedPaymentMethods}
                    selectedPaymentMethod={selectedPaymentMethod}
                  />
                ) : (
                  <div className="space-y-4">
                    <div className="flex items-center gap-2 rounded-lg border border-yellow-800 bg-yellow-950/50 p-4 text-sm text-yellow-400">
                      <AlertTriangle className="h-5 w-5 shrink-0" />
                      <div>
                        <p className="font-semibold">Stripe not configured</p>
                        <p className="text-yellow-400/80">
                          Please set NEXT_PUBLIC_STRIPE_PUBLISHABLE_KEY environment variable to enable payments.
                        </p>
                      </div>
                    </div>

                    <Button
                      disabled
                      className="w-full py-6 text-lg font-semibold bg-gray-700 text-gray-400 cursor-not-allowed"
                    >
                      <CreditCard className="mr-2 h-5 w-5" />
                      Payment Unavailable
                    </Button>
                  </div>
                )}

                {/* Error message */}
                {error && (
                  <div className="mt-4 p-3 bg-red-950/50 border border-red-800 rounded-lg">
                    <p className="text-sm text-red-400">{error}</p>
                  </div>
                )}

                {/* Cancel Button */}
                <Button
                  variant="ghost"
                  onClick={handleCancel}
                  className="mt-3 w-full text-gray-400 hover:text-gray-300"
                >
                  Cancel and Release Seats
                </Button>

                {/* Trust Badges */}
                <div className="mt-6 flex flex-wrap items-center justify-center gap-4 border-t border-gray-700 pt-6">
                  <div className="flex items-center gap-2 text-sm text-gray-400">
                    <Shield className="h-4 w-4 text-[#d4af37]" />
                    <span>SSL Encrypted</span>
                  </div>
                  <div className="flex items-center gap-2 text-sm text-gray-400">
                    <Lock className="h-4 w-4 text-[#d4af37]" />
                    <span>Secure Payment</span>
                  </div>
                  <div className="flex items-center gap-2 text-sm text-gray-400">
                    <CreditCard className="h-4 w-4 text-[#d4af37]" />
                    <span>PCI Compliant</span>
                  </div>
                </div>

                <p className="mt-4 text-center text-xs text-gray-500">
                  Your payment information is processed securely by Stripe. We never store your card details.
                </p>
              </div>
            </Card>
          </div>
        </div>
      </div>
    </div>
  )
}
