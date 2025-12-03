import { Clock, AlertTriangle } from "lucide-react"
import { cn } from "@/lib/utils"

interface CountdownTimerProps {
  timeRemaining: number
}

export function CountdownTimer({ timeRemaining }: CountdownTimerProps) {
  const minutes = Math.floor(timeRemaining / 60)
  const seconds = timeRemaining % 60
  const isUrgent = timeRemaining < 120

  return (
    <div
      className={cn(
        "flex items-center gap-2 rounded-lg border px-4 py-2 transition-colors",
        isUrgent
          ? "animate-pulse-urgent border-destructive bg-destructive/10 text-destructive"
          : "border-border bg-card text-card-foreground",
      )}
    >
      {isUrgent ? <AlertTriangle className="h-4 w-4" /> : <Clock className="h-4 w-4 text-muted-foreground" />}
      <div className="flex flex-col">
        <span className="text-xs text-muted-foreground">Session expires in</span>
        <span className={cn("font-mono text-lg font-bold", isUrgent && "text-destructive")}>
          {String(minutes).padStart(2, "0")}:{String(seconds).padStart(2, "0")}
        </span>
      </div>
    </div>
  )
}
