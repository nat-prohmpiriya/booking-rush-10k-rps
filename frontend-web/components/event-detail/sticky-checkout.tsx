"use client"

import { Button } from "@/components/ui/button"
import { ShoppingCart } from "lucide-react"
import Link from "next/link"

type StickyCheckoutProps = {
  eventId: string
  showId?: string
  selectedTickets: Record<string, number>
  totalPrice: number
  totalTickets: number
  isEventEnded?: boolean
  showStatus?: string // "scheduled" | "on_sale" | "sold_out" | "cancelled" | "completed"
}

export function StickyCheckout({
  eventId,
  showId,
  selectedTickets,
  totalPrice,
  totalTickets,
  isEventEnded = false,
  showStatus
}: StickyCheckoutProps) {
  // Only allow booking when status is "on_sale"
  const isSaleOpen = showStatus === "on_sale"
  const isDisabled = totalTickets === 0 || isEventEnded || !isSaleOpen

  // Determine button text and message based on status
  const getStatusMessage = () => {
    if (isEventEnded) return "This event has ended"
    if (showStatus === "scheduled") return "Sale not open yet"
    if (showStatus === "sold_out") return "Sold out"
    if (showStatus === "cancelled") return "Event cancelled"
    if (showStatus === "completed") return "Event completed"
    if (!showStatus) return "Sale not available"
    return null
  }

  const getButtonText = () => {
    if (isEventEnded) return "Event Ended"
    if (showStatus === "scheduled") return "Coming Soon"
    if (showStatus === "sold_out") return "Sold Out"
    if (showStatus === "cancelled") return "Cancelled"
    if (showStatus === "completed") return "Event Ended"
    if (!showStatus) return "Not Available"
    return "Reserve Now"
  }

  // Build queue URL with query params
  const buildQueueUrl = () => {
    if (isDisabled) return "#"
    const params = new URLSearchParams()
    params.set("event_id", eventId)
    if (showId) params.set("show_id", showId)
    params.set("tickets", JSON.stringify(selectedTickets))
    params.set("total", totalPrice.toString())
    return `/queue?${params.toString()}`
  }

  return (
    <div className="fixed bottom-0 left-0 right-0 border-t border-[#1a1a1a] bg-[#0a0a0a]/95 backdrop-blur-lg z-50">
      <div className="container mx-auto px-4 py-4">
        <div className="flex items-center justify-between max-w-4xl mx-auto">
          <div className="flex items-center gap-6">
            <div>
              <p className="text-sm text-muted-foreground">Total</p>
              <p className="text-3xl font-bold text-[#d4af37]">฿{totalPrice.toLocaleString()}</p>
            </div>
            {totalTickets > 0 && isSaleOpen && !isEventEnded && (
              <div className="hidden md:block text-sm text-muted-foreground">
                {totalTickets} {totalTickets === 1 ? "ticket" : "tickets"} selected
              </div>
            )}
            {getStatusMessage() && (
              <div className="hidden md:block text-sm text-amber-400">
                {getStatusMessage()}
              </div>
            )}
          </div>

          <Link href={buildQueueUrl()}>
            <Button
              size="lg"
              disabled={isDisabled}
              className="bg-[#d4af37] hover:bg-[#d4af37]/90 text-black font-semibold px-8 h-12 disabled:bg-zinc-600 disabled:text-zinc-400"
            >
              <ShoppingCart className="w-5 h-5 mr-2" />
              {getButtonText()}
            </Button>
          </Link>
        </div>
      </div>
    </div>
  )
}
