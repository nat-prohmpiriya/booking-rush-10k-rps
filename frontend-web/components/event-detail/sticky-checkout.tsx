"use client"

import { Button } from "@/components/ui/button"
import { ShoppingCart } from "lucide-react"
import Link from "next/link"

type StickyCheckoutProps = {
  totalPrice: number
  totalTickets: number
  isEventEnded?: boolean
}

export function StickyCheckout({ totalPrice, totalTickets, isEventEnded = false }: StickyCheckoutProps) {
  const isDisabled = totalTickets === 0 || isEventEnded

  return (
    <div className="fixed bottom-0 left-0 right-0 border-t border-[#1a1a1a] bg-[#0a0a0a]/95 backdrop-blur-lg z-50">
      <div className="container mx-auto px-4 py-4">
        <div className="flex items-center justify-between max-w-4xl mx-auto">
          <div className="flex items-center gap-6">
            <div>
              <p className="text-sm text-muted-foreground">Total</p>
              <p className="text-3xl font-bold text-[#d4af37]">฿{totalPrice.toLocaleString()}</p>
            </div>
            {totalTickets > 0 && !isEventEnded && (
              <div className="hidden md:block text-sm text-muted-foreground">
                {totalTickets} {totalTickets === 1 ? "ticket" : "tickets"} selected
              </div>
            )}
            {isEventEnded && (
              <div className="hidden md:block text-sm text-red-400">
                This event has ended
              </div>
            )}
          </div>

          <Link href={!isDisabled ? "/queue" : "#"}>
            <Button
              size="lg"
              disabled={isDisabled}
              className="bg-[#d4af37] hover:bg-[#d4af37]/90 text-black font-semibold px-8 h-12 disabled:bg-zinc-600 disabled:text-zinc-400"
            >
              <ShoppingCart className="w-5 h-5 mr-2" />
              {isEventEnded ? "Event Ended" : "Reserve Now"}
            </Button>
          </Link>
        </div>
      </div>
    </div>
  )
}
