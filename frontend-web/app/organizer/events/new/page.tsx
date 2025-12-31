"use client"

import { useState } from "react"
import { useRouter } from "next/navigation"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { ArrowLeft, Plus, Trash2, Calendar, Clock } from "lucide-react"
import Link from "next/link"
import { apiClient } from "@/lib/api/client"
import { DatePicker } from "@/components/ui/date-picker"
import { DateTimePicker } from "@/components/ui/datetime-picker"
import { TimePicker } from "@/components/ui/time-picker"
import { Textarea } from "@/components/ui/textarea"
import { format } from "date-fns"

interface ShowInput {
  name: string
  show_date: string
  start_time: string
  end_time: string
  zones: ZoneInput[]
}

interface ZoneInput {
  name: string
  price: number
  total_seats: number
  description: string
}

export default function CreateEventPage() {
  const router = useRouter()
  const [isSubmitting, setIsSubmitting] = useState(false)
  const [error, setError] = useState("")

  // Event form state
  const [eventData, setEventData] = useState({
    name: "",
    description: "",
    short_description: "",
    venue_name: "",
    venue_address: "",
    city: "",
    country: "Thailand",
    poster_url: "",
    banner_url: "",
    max_tickets_per_user: 4,
    booking_start_at: "",
    booking_end_at: "",
  })

  // Shows state
  const [shows, setShows] = useState<ShowInput[]>([
    {
      name: "Main Show",
      show_date: "",
      start_time: "19:00",
      end_time: "22:00",
      zones: [
        { name: "VIP", price: 5000, total_seats: 100, description: "VIP seating" },
        { name: "Standard", price: 2000, total_seats: 500, description: "Standard seating" },
      ],
    },
  ])

  const handleEventChange = (field: string, value: string | number) => {
    setEventData((prev) => ({ ...prev, [field]: value }))
  }

  const handleShowChange = (showIndex: number, field: string, value: string) => {
    setShows((prev) => {
      const updated = [...prev]
      updated[showIndex] = { ...updated[showIndex], [field]: value }
      return updated
    })
  }

  const handleZoneChange = (showIndex: number, zoneIndex: number, field: string, value: string | number) => {
    setShows((prev) => {
      const updated = [...prev]
      updated[showIndex].zones[zoneIndex] = {
        ...updated[showIndex].zones[zoneIndex],
        [field]: value,
      }
      return updated
    })
  }

  const addShow = () => {
    setShows((prev) => [
      ...prev,
      {
        name: `Show ${prev.length + 1}`,
        show_date: "",
        start_time: "19:00",
        end_time: "22:00",
        zones: [{ name: "Standard", price: 2000, total_seats: 100, description: "" }],
      },
    ])
  }

  const removeShow = (index: number) => {
    if (shows.length > 1) {
      setShows((prev) => prev.filter((_, i) => i !== index))
    }
  }

  const addZone = (showIndex: number) => {
    setShows((prev) => {
      const updated = [...prev]
      updated[showIndex].zones.push({
        name: `Zone ${updated[showIndex].zones.length + 1}`,
        price: 1000,
        total_seats: 100,
        description: "",
      })
      return updated
    })
  }

  const removeZone = (showIndex: number, zoneIndex: number) => {
    if (shows[showIndex].zones.length > 1) {
      setShows((prev) => {
        const updated = [...prev]
        updated[showIndex].zones = updated[showIndex].zones.filter((_, i) => i !== zoneIndex)
        return updated
      })
    }
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setError("")
    setIsSubmitting(true)

    try {
      // Validate
      if (!eventData.name) {
        throw new Error("Event name is required")
      }
      if (shows.length === 0) {
        throw new Error("At least one show is required")
      }
      for (const show of shows) {
        if (!show.show_date) {
          throw new Error("Show date is required for all shows")
        }
        if (show.zones.length === 0) {
          throw new Error("At least one zone is required per show")
        }
      }

      // Create event
      const eventPayload = {
        ...eventData,
        booking_start_at: eventData.booking_start_at ? new Date(eventData.booking_start_at).toISOString() : undefined,
        booking_end_at: eventData.booking_end_at ? new Date(eventData.booking_end_at).toISOString() : undefined,
      }

      const eventData2 = await apiClient.post<{ id: string }>("/events", eventPayload)
      const eventId = eventData2.id

      // Create shows and zones
      for (const show of shows) {
        const showPayload = {
          name: show.name,
          show_date: show.show_date,
          start_time: `${show.show_date}T${show.start_time}:00+07:00`,
          end_time: `${show.show_date}T${show.end_time}:00+07:00`,
        }

        const showData = await apiClient.post<{ id: string }>(`/events/${eventId}/shows`, showPayload)
        const showId = showData.id

        // Create zones for this show
        for (const zone of show.zones) {
          await apiClient.post(`/shows/${showId}/zones`, {
            name: zone.name,
            price: zone.price,
            total_seats: zone.total_seats,
            description: zone.description,
          })
        }
      }

      // Redirect to edit page for review before publishing
      router.push(`/organizer/events/${eventId}`)
    } catch (err) {
      console.error("Failed to create event:", err)
      setError(err instanceof Error ? err.message : "Failed to create event")
    } finally {
      setIsSubmitting(false)
    }
  }

  return (
    <div className="space-y-6 max-w-4xl" data-testid="organizer-new-event-page">
      {/* Page Header */}
      <div className="flex items-center gap-4">
        <Link href="/organizer/events">
          <Button variant="ghost" size="icon">
            <ArrowLeft className="h-4 w-4" />
          </Button>
        </Link>
        <div>
          <h1 className="text-3xl font-bold">Create Event</h1>
          <p className="text-muted-foreground mt-1">
            Fill in the details to create a new event
          </p>
        </div>
      </div>

      {error && (
        <div className="bg-destructive/10 text-destructive px-4 py-3 rounded-lg">
          {error}
        </div>
      )}

      <form onSubmit={handleSubmit} className="space-y-6" data-testid="organizer-new-event-form">
        {/* Basic Info */}
        <Card>
          <CardHeader>
            <CardTitle>Basic Information</CardTitle>
            <CardDescription>Enter the main details of your event</CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="grid gap-4 md:grid-cols-2">
              <div className="space-y-2">
                <Label htmlFor="name">Event Name *</Label>
                <Input
                  id="name"
                  value={eventData.name}
                  onChange={(e) => handleEventChange("name", e.target.value)}
                  placeholder="e.g., BTS World Tour"
                  required
                  data-testid="organizer-event-name-input"
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="city">City</Label>
                <Input
                  id="city"
                  value={eventData.city}
                  onChange={(e) => handleEventChange("city", e.target.value)}
                  placeholder="e.g., Bangkok"
                />
              </div>
            </div>

            <div className="space-y-2">
              <Label htmlFor="short_description">Short Description</Label>
              <Input
                id="short_description"
                value={eventData.short_description}
                onChange={(e) => handleEventChange("short_description", e.target.value)}
                placeholder="Brief description for listings"
              />
            </div>

            <div className="space-y-2">
              <Label htmlFor="description">Full Description</Label>
              <Textarea
                id="description"
                value={eventData.description}
                onChange={(e) => handleEventChange("description", e.target.value)}
                placeholder="Detailed description of your event"
                className="min-h-[100px]"
                data-testid="organizer-event-description-input"
              />
            </div>

            <div className="grid gap-4 md:grid-cols-2">
              <div className="space-y-2">
                <Label htmlFor="venue_name">Venue Name</Label>
                <Input
                  id="venue_name"
                  value={eventData.venue_name}
                  onChange={(e) => handleEventChange("venue_name", e.target.value)}
                  placeholder="e.g., Rajamangala Stadium"
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="venue_address">Venue Address</Label>
                <Input
                  id="venue_address"
                  value={eventData.venue_address}
                  onChange={(e) => handleEventChange("venue_address", e.target.value)}
                  placeholder="Full address"
                />
              </div>
            </div>

            <div className="grid gap-4 md:grid-cols-2">
              <div className="space-y-2">
                <Label htmlFor="poster_url">Poster Image URL</Label>
                <Input
                  id="poster_url"
                  value={eventData.poster_url}
                  onChange={(e) => handleEventChange("poster_url", e.target.value)}
                  placeholder="https://..."
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="max_tickets_per_user">Max Tickets Per User</Label>
                <Input
                  id="max_tickets_per_user"
                  type="number"
                  min={1}
                  value={eventData.max_tickets_per_user}
                  onChange={(e) => handleEventChange("max_tickets_per_user", parseInt(e.target.value) || 1)}
                />
              </div>
            </div>

            <div className="grid gap-4 md:grid-cols-2">
              <div className="space-y-2">
                <Label htmlFor="booking_start_at">Booking Start</Label>
                <DateTimePicker
                  value={eventData.booking_start_at}
                  onChange={(value) => handleEventChange("booking_start_at", value)}
                  placeholder="Select start date & time"
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="booking_end_at">Booking End</Label>
                <DateTimePicker
                  value={eventData.booking_end_at}
                  onChange={(value) => handleEventChange("booking_end_at", value)}
                  placeholder="Select end date & time"
                />
              </div>
            </div>
          </CardContent>
        </Card>

        {/* Shows & Zones */}
        <Card>
          <CardHeader className="flex flex-row items-center justify-between">
            <div>
              <CardTitle>Shows & Zones</CardTitle>
              <CardDescription>Configure show times and ticket zones</CardDescription>
            </div>
            <Button type="button" variant="outline" onClick={addShow}>
              <Plus className="h-4 w-4 mr-2" />
              Add Show
            </Button>
          </CardHeader>
          <CardContent className="space-y-6">
            {shows.map((show, showIndex) => (
              <div key={showIndex} className="border rounded-lg p-4 space-y-4">
                <div className="flex items-center justify-between">
                  <div className="flex items-center gap-2">
                    <Calendar className="h-4 w-4 text-muted-foreground" />
                    <span className="font-medium">Show {showIndex + 1}</span>
                  </div>
                  {shows.length > 1 && (
                    <Button
                      type="button"
                      variant="ghost"
                      size="sm"
                      onClick={() => removeShow(showIndex)}
                      className="text-destructive"
                    >
                      <Trash2 className="h-4 w-4" />
                    </Button>
                  )}
                </div>

                <div className="grid gap-4 md:grid-cols-4">
                  <div className="space-y-2">
                    <Label>Show Name</Label>
                    <Input
                      value={show.name}
                      onChange={(e) => handleShowChange(showIndex, "name", e.target.value)}
                      placeholder="e.g., Night 1"
                    />
                  </div>
                  <div className="space-y-2">
                    <Label>Date *</Label>
                    <DatePicker
                      value={show.show_date || ""}
                      onChange={(value) => handleShowChange(showIndex, "show_date", value)}
                      placeholder="Select date"
                    />
                  </div>
                  <div className="space-y-2">
                    <Label>Start Time</Label>
                    <TimePicker
                      value={show.start_time}
                      onChange={(value) => handleShowChange(showIndex, "start_time", value)}
                      placeholder="Start time"
                    />
                  </div>
                  <div className="space-y-2">
                    <Label>End Time</Label>
                    <TimePicker
                      value={show.end_time}
                      onChange={(value) => handleShowChange(showIndex, "end_time", value)}
                      placeholder="End time"
                    />
                  </div>
                </div>

                {/* Zones */}
                <div className="space-y-3">
                  <div className="flex items-center justify-between">
                    <Label className="text-sm text-muted-foreground">Ticket Zones</Label>
                    <Button type="button" variant="ghost" size="sm" onClick={() => addZone(showIndex)}>
                      <Plus className="h-3 w-3 mr-1" />
                      Add Zone
                    </Button>
                  </div>

                  {show.zones.map((zone, zoneIndex) => (
                    <div key={zoneIndex} className="grid gap-3 md:grid-cols-5 items-end bg-muted/50 p-3 rounded-lg">
                      <div className="space-y-1">
                        <Label className="text-xs">Zone Name</Label>
                        <Input
                          value={zone.name}
                          onChange={(e) => handleZoneChange(showIndex, zoneIndex, "name", e.target.value)}
                          placeholder="VIP"
                          className="h-9"
                        />
                      </div>
                      <div className="space-y-1">
                        <Label className="text-xs">Price (THB)</Label>
                        <Input
                          type="number"
                          min={0}
                          value={zone.price}
                          onChange={(e) => handleZoneChange(showIndex, zoneIndex, "price", parseInt(e.target.value) || 0)}
                          className="h-9"
                        />
                      </div>
                      <div className="space-y-1">
                        <Label className="text-xs">Total Seats</Label>
                        <Input
                          type="number"
                          min={1}
                          value={zone.total_seats}
                          onChange={(e) => handleZoneChange(showIndex, zoneIndex, "total_seats", parseInt(e.target.value) || 1)}
                          className="h-9"
                        />
                      </div>
                      <div className="space-y-1">
                        <Label className="text-xs">Description</Label>
                        <Input
                          value={zone.description}
                          onChange={(e) => handleZoneChange(showIndex, zoneIndex, "description", e.target.value)}
                          placeholder="Optional"
                          className="h-9"
                        />
                      </div>
                      <div>
                        {show.zones.length > 1 && (
                          <Button
                            type="button"
                            variant="ghost"
                            size="icon"
                            onClick={() => removeZone(showIndex, zoneIndex)}
                            className="h-9 w-9 text-destructive"
                          >
                            <Trash2 className="h-4 w-4" />
                          </Button>
                        )}
                      </div>
                    </div>
                  ))}
                </div>
              </div>
            ))}
          </CardContent>
        </Card>

        {/* Submit */}
        <div className="flex justify-end gap-4">
          <Link href="/organizer/events">
            <Button type="button" variant="outline">
              Cancel
            </Button>
          </Link>
          <Button type="submit" disabled={isSubmitting} data-testid="organizer-event-submit-button">
            {isSubmitting ? "Creating..." : "Create Event"}
          </Button>
        </div>
      </form>
    </div>
  )
}
