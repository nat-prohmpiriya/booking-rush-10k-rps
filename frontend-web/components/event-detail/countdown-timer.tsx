"use client"

import { useEffect, useState } from "react"
import { Card } from "@/components/ui/card"
import { Clock, CheckCircle } from "lucide-react"

interface CountdownTimerProps {
  targetDate?: Date
  saleEndDate?: Date
  showDate?: Date
  showStatus?: string // "scheduled" | "on_sale" | "sold_out" | "cancelled" | "completed"
}

export function CountdownTimer({ targetDate, saleEndDate, showDate, showStatus }: CountdownTimerProps) {
  const [timeLeft, setTimeLeft] = useState({
    days: 0,
    hours: 0,
    minutes: 0,
    seconds: 0,
  })
  const [saleState, setSaleState] = useState<"upcoming" | "open" | "ended" | "event_ended" | "sold_out" | "cancelled">("upcoming")

  useEffect(() => {
    const calculateTime = () => {
      const now = new Date().getTime()

      // Check if show date has passed (event already happened)
      if (showDate) {
        const showDateTime = showDate.getTime()
        // Add 24 hours to show date to account for the full day
        const showEndOfDay = showDateTime + (24 * 60 * 60 * 1000)
        if (now > showEndOfDay) {
          setSaleState("event_ended")
          return
        }
      }

      // Priority: Use showStatus from backend if available
      if (showStatus) {
        switch (showStatus) {
          case "on_sale":
            setSaleState("open")
            return
          case "sold_out":
            setSaleState("sold_out")
            return
          case "cancelled":
            setSaleState("cancelled")
            return
          case "completed":
            setSaleState("event_ended")
            return
          case "scheduled":
            // Show is scheduled but not on sale yet
            // If we have a targetDate, show countdown, otherwise show "Coming Soon"
            if (targetDate) {
              const saleStart = targetDate.getTime()
              if (now >= saleStart) {
                // Time has passed but backend still shows scheduled
                setSaleState("upcoming")
              } else {
                setSaleState("upcoming")
                const distance = saleStart - now
                setTimeLeft({
                  days: Math.floor(distance / (1000 * 60 * 60 * 24)),
                  hours: Math.floor((distance % (1000 * 60 * 60 * 24)) / (1000 * 60 * 60)),
                  minutes: Math.floor((distance % (1000 * 60 * 60)) / (1000 * 60)),
                  seconds: Math.floor((distance % (1000 * 60)) / 1000),
                })
              }
            } else {
              setSaleState("upcoming")
            }
            return
        }
      }

      // Fallback: Use dates if no showStatus
      if (!targetDate) {
        setSaleState("upcoming") // Default to upcoming if no info
        return
      }

      const saleStart = targetDate.getTime()
      const saleEnd = saleEndDate?.getTime()

      // Check if sale has ended
      if (saleEnd && now > saleEnd) {
        setSaleState("ended")
        return
      }

      // Check if sale is open
      if (now >= saleStart) {
        setSaleState("open")
        return
      }

      // Sale is upcoming - calculate countdown
      setSaleState("upcoming")
      const distance = saleStart - now

      setTimeLeft({
        days: Math.floor(distance / (1000 * 60 * 60 * 24)),
        hours: Math.floor((distance % (1000 * 60 * 60 * 24)) / (1000 * 60 * 60)),
        minutes: Math.floor((distance % (1000 * 60 * 60)) / (1000 * 60)),
        seconds: Math.floor((distance % (1000 * 60)) / 1000),
      })
    }

    calculateTime()
    const timer = setInterval(calculateTime, 1000)

    return () => clearInterval(timer)
  }, [targetDate, saleEndDate, showDate, showStatus])

  // Event has ended (show date passed)
  if (saleState === "event_ended") {
    return (
      <Card className="bg-zinc-500/10 border-zinc-500/30 p-6">
        <div className="flex items-center gap-4">
          <Clock className="w-6 h-6 text-zinc-500" />
          <div>
            <h3 className="text-lg font-semibold text-zinc-500">Event Ended</h3>
            <p className="text-sm text-muted-foreground">This event has already taken place</p>
          </div>
        </div>
      </Card>
    )
  }

  // Sale is open - show "Sale is Open" message
  if (saleState === "open") {
    return (
      <Card className="bg-green-500/10 border-green-500/30 p-6">
        <div className="flex items-center gap-4">
          <CheckCircle className="w-6 h-6 text-green-500" />
          <div>
            <h3 className="text-lg font-semibold text-green-500">Sale is Open!</h3>
            <p className="text-sm text-muted-foreground">Tickets are available for purchase</p>
          </div>
        </div>
      </Card>
    )
  }

  // Sold out
  if (saleState === "sold_out") {
    return (
      <Card className="bg-orange-500/10 border-orange-500/30 p-6">
        <div className="flex items-center gap-4">
          <Clock className="w-6 h-6 text-orange-500" />
          <div>
            <h3 className="text-lg font-semibold text-orange-500">Sold Out</h3>
            <p className="text-sm text-muted-foreground">All tickets have been sold</p>
          </div>
        </div>
      </Card>
    )
  }

  // Cancelled
  if (saleState === "cancelled") {
    return (
      <Card className="bg-red-500/10 border-red-500/30 p-6">
        <div className="flex items-center gap-4">
          <Clock className="w-6 h-6 text-red-500" />
          <div>
            <h3 className="text-lg font-semibold text-red-500">Event Cancelled</h3>
            <p className="text-sm text-muted-foreground">This event has been cancelled</p>
          </div>
        </div>
      </Card>
    )
  }

  // Sale has ended
  if (saleState === "ended") {
    return (
      <Card className="bg-red-500/10 border-red-500/30 p-6">
        <div className="flex items-center gap-4">
          <Clock className="w-6 h-6 text-red-500" />
          <div>
            <h3 className="text-lg font-semibold text-red-500">Sale Ended</h3>
            <p className="text-sm text-muted-foreground">Ticket sales have closed</p>
          </div>
        </div>
      </Card>
    )
  }

  // Sale is upcoming - show countdown or "Coming Soon"
  const hasCountdown = timeLeft.days > 0 || timeLeft.hours > 0 || timeLeft.minutes > 0 || timeLeft.seconds > 0

  if (!hasCountdown) {
    return (
      <Card className="bg-[#d4af37]/10 border-[#d4af37]/30 p-6">
        <div className="flex items-center gap-4">
          <Clock className="w-6 h-6 text-[#d4af37]" />
          <div>
            <h3 className="text-lg font-semibold text-[#d4af37]">Coming Soon</h3>
            <p className="text-sm text-muted-foreground">Ticket sales will open soon</p>
          </div>
        </div>
      </Card>
    )
  }

  return (
    <Card className="bg-[#d4af37]/10 border-[#d4af37]/30 p-6">
      <div className="flex items-center gap-4 mb-4">
        <Clock className="w-5 h-5 text-[#d4af37]" />
        <h3 className="text-lg font-semibold text-[#d4af37]">Sale Opens In</h3>
      </div>

      <div className="grid grid-cols-4 gap-4">
        <div className="text-center">
          <div className="text-3xl md:text-4xl font-bold text-white mb-1">{timeLeft.days}</div>
          <div className="text-sm text-muted-foreground">Days</div>
        </div>
        <div className="text-center">
          <div className="text-3xl md:text-4xl font-bold text-white mb-1">{timeLeft.hours}</div>
          <div className="text-sm text-muted-foreground">Hours</div>
        </div>
        <div className="text-center">
          <div className="text-3xl md:text-4xl font-bold text-white mb-1">{timeLeft.minutes}</div>
          <div className="text-sm text-muted-foreground">Minutes</div>
        </div>
        <div className="text-center">
          <div className="text-3xl md:text-4xl font-bold text-white mb-1">{timeLeft.seconds}</div>
          <div className="text-sm text-muted-foreground">Seconds</div>
        </div>
      </div>
    </Card>
  )
}
