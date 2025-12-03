import type { SeatZone } from "@/lib/seat-data"
import { cn } from "@/lib/utils"

interface SeatLegendProps {
  zones: SeatZone[]
}

export function SeatLegend({ zones }: SeatLegendProps) {
  return (
    <div className="flex flex-wrap items-center gap-4 rounded-lg border border-border bg-card p-4">
      <div className="flex items-center gap-2">
        <div className="h-4 w-4 rounded border-2 border-muted-foreground bg-transparent" />
        <span className="text-xs text-muted-foreground">Available</span>
      </div>
      <div className="flex items-center gap-2">
        <div className="h-4 w-4 rounded bg-primary" />
        <span className="text-xs text-muted-foreground">Selected</span>
      </div>
      <div className="flex items-center gap-2">
        <div className="h-4 w-4 rounded bg-muted" />
        <span className="text-xs text-muted-foreground">Sold</span>
      </div>
      <div className="ml-auto hidden items-center gap-4 lg:flex">
        {zones.map((zone) => (
          <div key={zone.id} className="flex items-center gap-2">
            <div className={cn("h-3 w-3 rounded-full", zone.colorClass)} />
            <span className="text-xs text-muted-foreground">
              {zone.name} (${zone.price})
            </span>
          </div>
        ))}
      </div>
    </div>
  )
}
