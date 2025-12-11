"use client"

import { useEffect, useState } from "react"
import { Card } from "@/components/ui/card"
import { Clock, CheckCircle } from "lucide-react"

interface CountdownTimerProps {
  targetDate?: Date
  saleEndDate?: Date
  showDate?: Date
}

export function CountdownTimer({ targetDate, saleEndDate, showDate }: CountdownTimerProps) {
  const [timeLeft, setTimeLeft] = useState({
    days: 0,
    hours: 0,
    minutes: 0,
    seconds: 0,
  })
  const [saleStatus, setSaleStatus] = useState<"upcoming" | "open" | "ended" | "event_ended">("upcoming")

  useEffect(() => {
    const calculateTime = () => {
      const now = new Date().getTime()

      // Check if show date has passed (event already happened)
      if (showDate) {
        const showDateTime = showDate.getTime()
        // Add 24 hours to show date to account for the full day
        const showEndOfDay = showDateTime + (24 * 60 * 60 * 1000)
        if (now > showEndOfDay) {
          setSaleStatus("event_ended")
          return
        }
      }

      // If no targetDate provided, check if event is still valid
      if (!targetDate) {
        setSaleStatus("open") // Assume sale is open if no date specified
        return
      }

      const saleStart = targetDate.getTime()
      const saleEnd = saleEndDate?.getTime()

      // Check if sale has ended
      if (saleEnd && now > saleEnd) {
        setSaleStatus("ended")
        return
      }

      // Check if sale is open
      if (now >= saleStart) {
        setSaleStatus("open")
        return
      }

      // Sale is upcoming - calculate countdown
      setSaleStatus("upcoming")
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
  }, [targetDate, saleEndDate, showDate])

  // Event has ended (show date passed)
  if (saleStatus === "event_ended") {
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
  if (saleStatus === "open") {
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

  // Sale has ended
  if (saleStatus === "ended") {
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

  // Sale is upcoming - show countdown
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
