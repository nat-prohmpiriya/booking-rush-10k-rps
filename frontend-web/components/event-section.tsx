"use client"

import { EventCard } from "@/components/event-card"
import type { EventDisplay } from "@/hooks/use-events"

interface EventSectionProps {
  title: string
  subtitle?: string
  badge?: string
  badgeVariant?: "primary" | "muted" | "warning"
  events: EventDisplay[]
  emptyMessage?: string
}

export function EventSection({
  title,
  subtitle,
  badge,
  badgeVariant = "primary",
  events,
  emptyMessage = "No events available",
}: EventSectionProps) {
  if (events.length === 0) {
    return null
  }

  const badgeColors = {
    primary: "text-primary",
    muted: "text-muted-foreground",
    warning: "text-amber-500",
  }

  return (
    <section className="space-y-8">
      <div className="space-y-4">
        <h2 className="text-2xl lg:text-4xl font-bold text-balance text-primary uppercase">{title}</h2>
        {subtitle && (
          <p className="text-primary max-w-2xl text-pretty font-semibold">
            {subtitle}
          </p>
        )}
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6 lg:gap-8">
        {events.map((event) => (
          <EventCard
            key={event.id}
            id={event.id}
            title={event.title}
            venue={event.venue}
            date={event.date}
            price={event.price}
            image={event.image}
          />
        ))}
      </div>
    </section>
  )
}
