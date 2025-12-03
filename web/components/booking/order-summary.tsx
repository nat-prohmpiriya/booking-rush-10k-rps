"use client"

import { X, Ticket, CreditCard, ShieldCheck } from "lucide-react"
import { Button } from "@/components/ui/button"
import { Separator } from "@/components/ui/separator"
import type { Seat, SeatZone } from "@/lib/seat-data"
import { cn } from "@/lib/utils"

interface OrderSummaryProps {
  selectedSeats: Seat[]
  zones: SeatZone[]
  totalPrice: number
  onRemoveSeat: (seatId: string) => void
  timeRemaining: number
}

export function OrderSummary({ selectedSeats, zones, totalPrice, onRemoveSeat, timeRemaining }: OrderSummaryProps) {
  const serviceFee = selectedSeats.length * 15
  const grandTotal = totalPrice + serviceFee

  return (
    <div className="rounded-lg border border-border bg-card">
      <div className="p-6">
        <h2 className="flex items-center gap-2 text-lg font-semibold text-card-foreground">
          <Ticket className="h-5 w-5" />
          Order Summary
        </h2>

        {selectedSeats.length === 0 ? (
          <div className="mt-6 rounded-lg border-2 border-dashed border-border p-8 text-center">
            <p className="text-sm text-muted-foreground">Select seats from the map to begin</p>
          </div>
        ) : (
          <div className="mt-4 space-y-3">
            {selectedSeats.map((seat) => {
              const zone = zones.find((z) => z.id === seat.zone)
              return (
                <div key={seat.id} className="flex items-center justify-between rounded-lg bg-muted/50 p-3">
                  <div className="flex items-center gap-3">
                    <div
                      className={cn(
                        "flex h-8 w-8 items-center justify-center rounded text-xs font-bold",
                        zone?.colorClass,
                        "text-primary-foreground",
                      )}
                    >
                      {seat.row}
                      {seat.number}
                    </div>
                    <div>
                      <p className="text-sm font-medium text-card-foreground">
                        Row {seat.row}, Seat {seat.number}
                      </p>
                      <p className="text-xs text-muted-foreground">{zone?.name}</p>
                    </div>
                  </div>
                  <div className="flex items-center gap-2">
                    <span className="text-sm font-semibold text-card-foreground">${zone?.price}</span>
                    <Button
                      variant="ghost"
                      size="icon"
                      className="h-6 w-6 text-muted-foreground hover:text-destructive"
                      onClick={() => onRemoveSeat(seat.id)}
                    >
                      <X className="h-3 w-3" />
                    </Button>
                  </div>
                </div>
              )
            })}
          </div>
        )}
      </div>

      {selectedSeats.length > 0 && (
        <>
          <Separator />
          <div className="space-y-3 p-6">
            <div className="flex justify-between text-sm">
              <span className="text-muted-foreground">Tickets ({selectedSeats.length})</span>
              <span className="text-card-foreground">${totalPrice.toFixed(2)}</span>
            </div>
            <div className="flex justify-between text-sm">
              <span className="text-muted-foreground">Service Fee</span>
              <span className="text-card-foreground">${serviceFee.toFixed(2)}</span>
            </div>
            <Separator />
            <div className="flex justify-between">
              <span className="font-semibold text-card-foreground">Total</span>
              <span className="text-xl font-bold text-card-foreground">${grandTotal.toFixed(2)}</span>
            </div>
          </div>
          <div className="p-6 pt-0">
            <Button
              className="w-full gap-2 bg-primary text-primary-foreground hover:bg-primary/90"
              size="lg"
              disabled={timeRemaining === 0}
            >
              <CreditCard className="h-4 w-4" />
              Proceed to Checkout
            </Button>
            <div className="mt-3 flex items-center justify-center gap-2 text-xs text-muted-foreground">
              <ShieldCheck className="h-3 w-3" />
              <span>Secure checkout • 100% money-back guarantee</span>
            </div>
          </div>
        </>
      )}

      {/* Mobile zone legend */}
      <div className="border-t border-border p-4 lg:hidden">
        <p className="mb-2 text-xs font-medium text-muted-foreground">Pricing Zones</p>
        <div className="grid grid-cols-2 gap-2">
          {zones.map((zone) => (
            <div key={zone.id} className="flex items-center gap-2">
              <div className={cn("h-3 w-3 rounded-full", zone.colorClass)} />
              <span className="text-xs text-muted-foreground">
                {zone.name}: ${zone.price}
              </span>
            </div>
          ))}
        </div>
      </div>
    </div>
  )
}
