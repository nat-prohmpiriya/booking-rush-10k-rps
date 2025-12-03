"use client"

import { useState, useEffect, useCallback } from "react"
import { BookingHeader } from "./booking/booking-header"
import { CountdownTimer } from "./booking/countdown-timer"
import { SeatMap } from "./booking/seat-map"
import { SeatLegend } from "./booking/seat-legend"
import { OrderSummary } from "./booking/order-summary"
import { AvailabilityBanner } from "./booking/availability-banner"
import type { Seat } from "@/lib/seat-data"
import { generateSeats, ZONES } from "@/lib/seat-data"

export default function SeatSelectionPage() {
  const [seats, setSeats] = useState<Seat[]>([])
  const [selectedSeats, setSelectedSeats] = useState<Seat[]>([])
  const [timeRemaining, setTimeRemaining] = useState(600) // 10 minutes
  const [availableCount, setAvailableCount] = useState(0)
  const [recentlyTaken, setRecentlyTaken] = useState<string[]>([])

  // Initialize seats
  useEffect(() => {
    const initialSeats = generateSeats()
    setSeats(initialSeats)
    setAvailableCount(initialSeats.filter((s) => s.status === "available").length)
  }, [])

  // Countdown timer
  useEffect(() => {
    const timer = setInterval(() => {
      setTimeRemaining((prev) => (prev > 0 ? prev - 1 : 0))
    }, 1000)
    return () => clearInterval(timer)
  }, [])

  // Simulate real-time seat availability changes
  useEffect(() => {
    const interval = setInterval(() => {
      setSeats((prevSeats) => {
        const availableSeats = prevSeats.filter(
          (s) => s.status === "available" && !selectedSeats.some((sel) => sel.id === s.id),
        )
        if (availableSeats.length > 5 && Math.random() > 0.6) {
          const randomIndex = Math.floor(Math.random() * availableSeats.length)
          const seatToTake = availableSeats[randomIndex]
          setRecentlyTaken((prev) => [...prev.slice(-2), seatToTake.id])
          setTimeout(() => {
            setRecentlyTaken((prev) => prev.filter((id) => id !== seatToTake.id))
          }, 3000)
          return prevSeats.map((s) => (s.id === seatToTake.id ? { ...s, status: "sold" as const } : s))
        }
        return prevSeats
      })
    }, 4000)
    return () => clearInterval(interval)
  }, [selectedSeats])

  // Update available count
  useEffect(() => {
    setAvailableCount(seats.filter((s) => s.status === "available").length)
  }, [seats])

  const handleSeatClick = useCallback((seat: Seat) => {
    if (seat.status !== "available") return

    setSelectedSeats((prev) => {
      const isSelected = prev.some((s) => s.id === seat.id)
      if (isSelected) {
        return prev.filter((s) => s.id !== seat.id)
      }
      if (prev.length >= 6) return prev // Max 6 seats
      return [...prev, seat]
    })
  }, [])

  const handleRemoveSeat = useCallback((seatId: string) => {
    setSelectedSeats((prev) => prev.filter((s) => s.id !== seatId))
  }, [])

  const totalPrice = selectedSeats.reduce((sum, seat) => {
    const zone = ZONES.find((z) => z.id === seat.zone)
    return sum + (zone?.price || 0)
  }, 0)

  return (
    <div className="min-h-screen bg-background">
      <BookingHeader />

      <main className="container mx-auto px-4 py-6 lg:py-8">
        <div className="mb-6 flex flex-col gap-4 lg:flex-row lg:items-center lg:justify-between">
          <div>
            <h1 className="text-2xl font-bold tracking-tight text-foreground lg:text-3xl">Select Your Seats</h1>
            <p className="mt-1 text-muted-foreground">Neon Nights World Tour • Madison Square Garden • Dec 15, 2025</p>
          </div>
          <CountdownTimer timeRemaining={timeRemaining} />
        </div>

        <AvailabilityBanner availableCount={availableCount} recentlyTaken={recentlyTaken} />

        <div className="mt-6 grid gap-6 lg:grid-cols-[1fr_380px]">
          <div className="space-y-4">
            <SeatLegend zones={ZONES} />
            <SeatMap
              seats={seats}
              selectedSeats={selectedSeats}
              onSeatClick={handleSeatClick}
              recentlyTaken={recentlyTaken}
            />
          </div>

          <div className="lg:sticky lg:top-6 lg:self-start">
            <OrderSummary
              selectedSeats={selectedSeats}
              zones={ZONES}
              totalPrice={totalPrice}
              onRemoveSeat={handleRemoveSeat}
              timeRemaining={timeRemaining}
            />
          </div>
        </div>
      </main>
    </div>
  )
}
