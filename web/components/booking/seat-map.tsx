"use client"

import { useRef, useState } from "react"
import type { Seat } from "@/lib/seat-data"
import { ZONES } from "@/lib/seat-data"
import { cn } from "@/lib/utils"
import { ZoomIn, ZoomOut, RotateCcw } from "lucide-react"
import { Button } from "@/components/ui/button"
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@/components/ui/tooltip"

interface SeatMapProps {
  seats: Seat[]
  selectedSeats: Seat[]
  onSeatClick: (seat: Seat) => void
  recentlyTaken: string[]
}

export function SeatMap({ seats, selectedSeats, onSeatClick, recentlyTaken }: SeatMapProps) {
  const [zoom, setZoom] = useState(1)
  const [pan, setPan] = useState({ x: 0, y: 0 })
  const containerRef = useRef<HTMLDivElement>(null)

  const handleZoomIn = () => setZoom((prev) => Math.min(prev + 0.25, 2))
  const handleZoomOut = () => setZoom((prev) => Math.max(prev - 0.25, 0.5))
  const handleReset = () => {
    setZoom(1)
    setPan({ x: 0, y: 0 })
  }

  // Group seats by row
  const seatsByRow = seats.reduce(
    (acc, seat) => {
      if (!acc[seat.row]) acc[seat.row] = []
      acc[seat.row].push(seat)
      return acc
    },
    {} as Record<string, Seat[]>,
  )

  const rows = Object.keys(seatsByRow).sort()

  return (
    <div className="relative overflow-hidden rounded-lg border border-border bg-card">
      {/* Zoom controls */}
      <div className="absolute right-4 top-4 z-10 flex flex-col gap-1">
        <Button variant="secondary" size="icon" onClick={handleZoomIn} className="h-8 w-8">
          <ZoomIn className="h-4 w-4" />
        </Button>
        <Button variant="secondary" size="icon" onClick={handleZoomOut} className="h-8 w-8">
          <ZoomOut className="h-4 w-4" />
        </Button>
        <Button variant="secondary" size="icon" onClick={handleReset} className="h-8 w-8">
          <RotateCcw className="h-4 w-4" />
        </Button>
      </div>

      <div ref={containerRef} className="overflow-auto p-6" style={{ maxHeight: "600px" }}>
        <div
          className="mx-auto transition-transform duration-200"
          style={{
            transform: `scale(${zoom}) translate(${pan.x}px, ${pan.y}px)`,
            transformOrigin: "center top",
          }}
        >
          {/* Stage */}
          <div className="mx-auto mb-8 flex h-16 w-3/4 items-center justify-center rounded-t-full bg-gradient-to-b from-muted to-transparent">
            <span className="text-sm font-medium uppercase tracking-widest text-muted-foreground">Stage</span>
          </div>

          {/* Seat grid */}
          <TooltipProvider delayDuration={0}>
            <div className="flex flex-col items-center gap-1">
              {rows.map((row) => (
                <div key={row} className="flex items-center gap-1">
                  <span className="w-6 text-right text-xs text-muted-foreground">{row}</span>
                  <div className="flex gap-1">
                    {seatsByRow[row]
                      .sort((a, b) => a.number - b.number)
                      .map((seat) => {
                        const isSelected = selectedSeats.some((s) => s.id === seat.id)
                        const isRecentlyTaken = recentlyTaken.includes(seat.id)
                        const zone = ZONES.find((z) => z.id === seat.zone)

                        return (
                          <Tooltip key={seat.id}>
                            <TooltipTrigger asChild>
                              <button
                                onClick={() => onSeatClick(seat)}
                                disabled={seat.status !== "available"}
                                className={cn(
                                  "relative h-6 w-6 rounded text-[10px] font-medium transition-all duration-200 focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2 focus:ring-offset-background",
                                  seat.status === "available" &&
                                    !isSelected && [
                                      "border-2 hover:scale-110",
                                      zone?.borderClass,
                                      "bg-transparent hover:bg-muted/50",
                                    ],
                                  seat.status === "available" &&
                                    isSelected && ["bg-primary text-primary-foreground scale-110 shadow-lg"],
                                  seat.status === "sold" && "cursor-not-allowed bg-muted text-muted-foreground/50",
                                  isRecentlyTaken && "animate-pulse bg-destructive/50",
                                )}
                                aria-label={`Row ${seat.row}, Seat ${seat.number}, ${zone?.name}, $${zone?.price}`}
                              >
                                {isSelected ? "✓" : seat.number}
                              </button>
                            </TooltipTrigger>
                            <TooltipContent side="top" className="bg-popover text-popover-foreground">
                              <div className="text-center">
                                <p className="font-medium">
                                  Row {seat.row}, Seat {seat.number}
                                </p>
                                <p className="text-xs text-muted-foreground">
                                  {zone?.name} • ${zone?.price}
                                </p>
                                {seat.status === "sold" && <p className="mt-1 text-xs text-destructive">Sold</p>}
                              </div>
                            </TooltipContent>
                          </Tooltip>
                        )
                      })}
                  </div>
                  <span className="w-6 text-left text-xs text-muted-foreground">{row}</span>
                </div>
              ))}
            </div>
          </TooltipProvider>

          {/* Section labels */}
          <div className="mt-8 flex justify-center gap-8">
            <span className="text-xs text-muted-foreground">← Left</span>
            <span className="text-xs text-muted-foreground">Center</span>
            <span className="text-xs text-muted-foreground">Right →</span>
          </div>
        </div>
      </div>
    </div>
  )
}
