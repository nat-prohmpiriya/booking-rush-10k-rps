import { Users, Zap } from "lucide-react"
import { cn } from "@/lib/utils"

interface AvailabilityBannerProps {
  availableCount: number
  recentlyTaken: string[]
}

export function AvailabilityBanner({ availableCount, recentlyTaken }: AvailabilityBannerProps) {
  const isLowAvailability = availableCount < 50

  return (
    <div
      className={cn(
        "flex flex-wrap items-center justify-between gap-4 rounded-lg border p-4",
        isLowAvailability ? "border-destructive/50 bg-destructive/5" : "border-border bg-card",
      )}
    >
      <div className="flex items-center gap-3">
        <div
          className={cn(
            "flex h-10 w-10 items-center justify-center rounded-full",
            isLowAvailability ? "bg-destructive/20" : "bg-muted",
          )}
        >
          <Users className={cn("h-5 w-5", isLowAvailability ? "text-destructive" : "text-muted-foreground")} />
        </div>
        <div>
          <p className="text-sm font-medium text-foreground">{availableCount} seats remaining</p>
          <p className="text-xs text-muted-foreground">
            {isLowAvailability ? "Selling fast! Don't miss out." : "Select up to 6 seats per order"}
          </p>
        </div>
      </div>

      {recentlyTaken.length > 0 && (
        <div className="flex items-center gap-2 rounded-full bg-destructive/10 px-3 py-1.5 text-xs text-destructive">
          <Zap className="h-3 w-3" />
          <span>Seat just taken by another user</span>
        </div>
      )}

      <div className="flex items-center gap-2 text-xs text-muted-foreground">
        <span className="inline-block h-2 w-2 animate-pulse rounded-full bg-green-500" />
        <span>Live availability</span>
      </div>
    </div>
  )
}
