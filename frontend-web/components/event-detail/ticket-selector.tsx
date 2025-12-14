"use client"

import { Minus, Plus } from "lucide-react"
import { Card } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"

export type TicketZone = {
  id: string
  name: string
  price: number
  available: number
  soldOut: boolean
  maxPerOrder?: number
  minPerOrder?: number
}

export type BookingSummary = {
  bookedCount: number
  maxAllowed: number
  remainingSlots: number
}

type TicketSelectorProps = {
  zones: TicketZone[]
  selectedTickets: Record<string, number>
  onTicketChange: (zoneId: string, quantity: number) => void
  bookingSummary?: BookingSummary | null
}

export function TicketSelector({ zones, selectedTickets, onTicketChange, bookingSummary }: TicketSelectorProps) {
  // Calculate total selected tickets
  const totalSelected = Object.values(selectedTickets).reduce((sum, qty) => sum + qty, 0)

  // Calculate remaining slots considering user's existing bookings
  const userBookedCount = bookingSummary?.bookedCount || 0
  const maxAllowedPerUser = bookingSummary?.maxAllowed || 10
  const userRemainingSlots = bookingSummary?.remainingSlots ?? maxAllowedPerUser

  return (
    <div className="space-y-6">
      <div>
        <h2 className="text-3xl font-bold mb-2">Select Tickets</h2>
        <p className="text-muted-foreground">Choose your preferred seating zone</p>

        {/* Show user's booking summary if they have existing bookings */}
        {bookingSummary && userBookedCount > 0 && (
          <div className="mt-3 p-3 bg-zinc-800/50 rounded-lg border border-zinc-700">
            <p className="text-sm text-zinc-300">
              You have booked <span className="font-semibold text-[#d4af37]">{userBookedCount}</span> of{" "}
              <span className="font-semibold">{maxAllowedPerUser}</span> tickets for this event.
              {userRemainingSlots > 0 ? (
                <span className="text-zinc-400"> ({userRemainingSlots} remaining)</span>
              ) : (
                <span className="text-red-400"> (limit reached)</span>
              )}
            </p>
          </div>
        )}
      </div>

      <div className="grid gap-4">
        {zones.map((zone) => {
          const maxPerOrder = zone.maxPerOrder || 10
          const minRequired = zone.minPerOrder || 1
          const currentQty = selectedTickets[zone.id] || 0

          // Calculate the effective max: min of (zone max, available seats, user remaining slots)
          const effectiveMax = Math.min(
            maxPerOrder,
            zone.available,
            Math.max(0, userRemainingSlots - totalSelected + currentQty) // Slots available for this zone
          )

          const isAtUserLimit = userRemainingSlots <= 0 && currentQty === 0
          const cannotAddMore = currentQty >= effectiveMax

          return (
            <Card
              key={zone.id}
              className={`p-6 transition-all ${
                zone.soldOut || isAtUserLimit
                  ? "bg-[#0f0f0f] border-[#1a1a1a] opacity-50"
                  : "bg-[#0f0f0f] border-[#1a1a1a] hover:border-[#d4af37]/50"
              }`}
            >
              <div className="flex flex-col md:flex-row md:items-center justify-between gap-4">
                <div className="flex-1">
                  <div className="flex items-center gap-3 mb-2">
                    <h3 className="text-xl font-semibold">{zone.name}</h3>
                    {zone.soldOut && (
                      <Badge variant="secondary" className="bg-red-500/10 text-red-500 border-red-500/20">
                        Sold Out
                      </Badge>
                    )}
                    {!zone.soldOut && isAtUserLimit && (
                      <Badge variant="secondary" className="bg-orange-500/10 text-orange-500 border-orange-500/20">
                        Limit Reached
                      </Badge>
                    )}
                  </div>
                  <div className="flex items-baseline gap-3 flex-wrap">
                    <p className="text-2xl font-bold text-[#d4af37]">฿{zone.price.toLocaleString()}</p>
                    {!zone.soldOut && (
                      <>
                        <p className="text-sm text-muted-foreground">{zone.available} tickets remaining</p>
                        <span className="text-sm text-zinc-500">•</span>
                        <p className="text-sm text-zinc-400">Max {maxPerOrder} per order</p>
                      </>
                    )}
                  </div>
                </div>

                {!zone.soldOut && !isAtUserLimit && (
                  <div className="flex items-center gap-3">
                    <Button
                      variant="outline"
                      size="icon"
                      onClick={() => onTicketChange(zone.id, Math.max(0, currentQty - 1))}
                      disabled={currentQty === 0}
                      className="h-10 w-10 rounded-full border-[#d4af37]/30 hover:bg-[#d4af37]/10 hover:border-[#d4af37]"
                    >
                      <Minus className="h-4 w-4 text-[#d4af37]" />
                    </Button>

                    <div className="w-12 text-center">
                      <span className="text-xl font-semibold">{currentQty}</span>
                    </div>

                    <Button
                      variant="outline"
                      size="icon"
                      onClick={() => onTicketChange(zone.id, Math.min(effectiveMax, currentQty + 1))}
                      disabled={cannotAddMore}
                      className="h-10 w-10 rounded-full border-[#d4af37]/30 hover:bg-[#d4af37]/10 hover:border-[#d4af37]"
                    >
                      <Plus className="h-4 w-4 text-[#d4af37]" />
                    </Button>
                  </div>
                )}
              </div>
            </Card>
          )
        })}
      </div>
    </div>
  )
}
