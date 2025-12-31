"use client"

import { useEffect, useState } from "react"
import Link from "next/link"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Calendar, Ticket, DollarSign, Users, Plus, ArrowRight } from "lucide-react"
import { eventsApi } from "@/lib/api"
import type { EventResponse } from "@/lib/api/types"

interface DashboardStats {
  totalEvents: number
  activeEvents: number
  totalBookings: number
  totalRevenue: number
}

export default function OrganizerDashboard() {
  const [events, setEvents] = useState<EventResponse[]>([])
  const [stats, setStats] = useState<DashboardStats>({
    totalEvents: 0,
    activeEvents: 0,
    totalBookings: 0,
    totalRevenue: 0,
  })
  const [isLoading, setIsLoading] = useState(true)

  useEffect(() => {
    loadDashboardData()
  }, [])

  const loadDashboardData = async () => {
    try {
      setIsLoading(true)
      const response = await eventsApi.list({ limit: 5 })
      const eventList = response.data || []
      setEvents(eventList)

      // Calculate stats
      const activeCount = eventList.filter(
        (e) => e.status === "published" || e.status === "on_sale"
      ).length

      setStats({
        totalEvents: eventList.length,
        activeEvents: activeCount,
        totalBookings: 0, // TODO: Get from API
        totalRevenue: 0, // TODO: Get from API
      })
    } catch (error) {
      console.error("Failed to load dashboard data:", error)
    } finally {
      setIsLoading(false)
    }
  }

  const statCards = [
    {
      title: "Total Events",
      value: stats.totalEvents,
      icon: Calendar,
      color: "text-blue-500",
      bgColor: "bg-blue-500/10",
    },
    {
      title: "Active Events",
      value: stats.activeEvents,
      icon: Ticket,
      color: "text-green-500",
      bgColor: "bg-green-500/10",
    },
    {
      title: "Total Bookings",
      value: stats.totalBookings,
      icon: Users,
      color: "text-purple-500",
      bgColor: "bg-purple-500/10",
    },
    {
      title: "Total Revenue",
      value: `${stats.totalRevenue.toLocaleString()} THB`,
      icon: DollarSign,
      color: "text-amber-500",
      bgColor: "bg-amber-500/10",
    },
  ]

  return (
    <div className="space-y-8" data-testid="organizer-dashboard-page">
      {/* Page Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold">Dashboard</h1>
          <p className="text-muted-foreground mt-1">
            Welcome back! Here&apos;s an overview of your events.
          </p>
        </div>
        <Link href="/organizer/events/new">
          <Button data-testid="organizer-create-event-button">
            <Plus className="h-4 w-4 mr-2" />
            Create Event
          </Button>
        </Link>
      </div>

      {/* Stats Grid */}
      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4" data-testid="organizer-stats">
        {statCards.map((stat) => (
          <Card key={stat.title}>
            <CardHeader className="flex flex-row items-center justify-between pb-2">
              <CardTitle className="text-sm font-medium text-muted-foreground">
                {stat.title}
              </CardTitle>
              <div className={`p-2 rounded-lg ${stat.bgColor}`}>
                <stat.icon className={`h-4 w-4 ${stat.color}`} />
              </div>
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold">
                {isLoading ? (
                  <div className="h-8 w-20 bg-muted animate-pulse rounded" />
                ) : (
                  stat.value
                )}
              </div>
            </CardContent>
          </Card>
        ))}
      </div>

      {/* Recent Events */}
      <Card data-testid="organizer-events-section">
        <CardHeader className="flex flex-row items-center justify-between">
          <div>
            <CardTitle>Recent Events</CardTitle>
            <CardDescription>Your latest created events</CardDescription>
          </div>
          <Link href="/organizer/events">
            <Button variant="outline" size="sm">
              View All
              <ArrowRight className="h-4 w-4 ml-2" />
            </Button>
          </Link>
        </CardHeader>
        <CardContent>
          {isLoading ? (
            <div className="space-y-4">
              {[1, 2, 3].map((i) => (
                <div key={i} className="flex items-center gap-4">
                  <div className="h-12 w-12 bg-muted animate-pulse rounded" />
                  <div className="flex-1 space-y-2">
                    <div className="h-4 w-48 bg-muted animate-pulse rounded" />
                    <div className="h-3 w-32 bg-muted animate-pulse rounded" />
                  </div>
                </div>
              ))}
            </div>
          ) : events.length === 0 ? (
            <div className="text-center py-8">
              <Calendar className="h-12 w-12 mx-auto text-muted-foreground mb-4" />
              <h3 className="font-medium mb-2">No events yet</h3>
              <p className="text-sm text-muted-foreground mb-4">
                Create your first event to get started
              </p>
              <Link href="/organizer/events/new">
                <Button>
                  <Plus className="h-4 w-4 mr-2" />
                  Create Event
                </Button>
              </Link>
            </div>
          ) : (
            <div className="space-y-4">
              {events.map((event) => (
                <Link
                  key={event.id}
                  href={`/organizer/events/${event.id}`}
                  className="flex items-center gap-4 p-3 rounded-lg hover:bg-muted transition-colors"
                >
                  {event.poster_url ? (
                    <img
                      src={event.poster_url}
                      alt={event.name}
                      className="h-12 w-12 rounded object-cover"
                    />
                  ) : (
                    <div className="h-12 w-12 rounded bg-muted flex items-center justify-center">
                      <Calendar className="h-6 w-6 text-muted-foreground" />
                    </div>
                  )}
                  <div className="flex-1 min-w-0">
                    <p className="font-medium truncate">{event.name}</p>
                    <p className="text-sm text-muted-foreground">
                      {event.city} &bull; {event.status}
                    </p>
                  </div>
                  <ArrowRight className="h-4 w-4 text-muted-foreground" />
                </Link>
              ))}
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  )
}
