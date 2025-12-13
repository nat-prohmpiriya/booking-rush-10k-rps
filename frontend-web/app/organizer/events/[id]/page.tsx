"use client"

import { useEffect, useState } from "react"
import { useParams, useRouter } from "next/navigation"
import Link from "next/link"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Badge } from "@/components/ui/badge"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import {
  ArrowLeft,
  Save,
  Calendar,
  Ticket,
  Settings,
  Trash2,
  Plus,
  Edit,
  Eye,
  CheckCircle,
  Clock,
  AlertCircle,
  DollarSign,
  Users,
  TrendingUp,
  BarChart3,
} from "lucide-react"
import { Switch } from "@/components/ui/switch"
import { eventsApi, showsApi, zonesApi, UpdateEventRequest, UpdateShowRequest, UpdateZoneRequest } from "@/lib/api"
import type { EventResponse, ShowResponse, ShowZoneResponse } from "@/lib/api/types"

export default function EditEventPage() {
  const params = useParams()
  const router = useRouter()
  const eventId = params.id as string

  const [event, setEvent] = useState<EventResponse | null>(null)
  const [shows, setShows] = useState<ShowResponse[]>([])
  const [zonesByShow, setZonesByShow] = useState<Record<string, ShowZoneResponse[]>>({})
  const [isLoading, setIsLoading] = useState(true)
  const [isSaving, setIsSaving] = useState(false)
  const [error, setError] = useState("")
  const [successMessage, setSuccessMessage] = useState("")
  const [activeTab, setActiveTab] = useState("details")

  // Edit form state
  const [editForm, setEditForm] = useState<UpdateEventRequest>({})
  const [editingShow, setEditingShow] = useState<string | null>(null)
  const [editingZone, setEditingZone] = useState<string | null>(null)
  const [showForm, setShowForm] = useState<UpdateShowRequest>({})
  const [zoneForm, setZoneForm] = useState<UpdateZoneRequest>({})

  useEffect(() => {
    if (eventId) {
      loadEventData()
    }
  }, [eventId])

  const loadEventData = async () => {
    try {
      setIsLoading(true)
      setError("")

      // Load event
      const eventData = await eventsApi.getById(eventId)
      setEvent(eventData)
      setEditForm({
        name: eventData.name,
        description: eventData.description,
        short_description: eventData.short_description,
        poster_url: eventData.poster_url,
        banner_url: eventData.banner_url,
        venue_name: eventData.venue_name,
        venue_address: eventData.venue_address,
        city: eventData.city,
        country: eventData.country,
        max_tickets_per_user: eventData.max_tickets_per_user,
        is_featured: eventData.is_featured,
        is_public: eventData.is_public,
      })

      // Load shows
      const showsData = await eventsApi.getEventShows(eventId)
      setShows(showsData)

      // Load zones for each show
      const zonesMap: Record<string, ShowZoneResponse[]> = {}
      for (const show of showsData) {
        const zones = await eventsApi.getShowZones(show.id)
        zonesMap[show.id] = zones
      }
      setZonesByShow(zonesMap)
    } catch (err) {
      console.error("Failed to load event:", err)
      setError("Failed to load event data")
    } finally {
      setIsLoading(false)
    }
  }

  const handleSaveEvent = async () => {
    if (!event) return

    try {
      setIsSaving(true)
      setError("")
      setSuccessMessage("")

      const updatedEvent = await eventsApi.update(event.id, editForm)
      setEvent(updatedEvent)
      setSuccessMessage("Event updated successfully!")

      setTimeout(() => setSuccessMessage(""), 3000)
    } catch (err) {
      console.error("Failed to update event:", err)
      setError("Failed to update event")
    } finally {
      setIsSaving(false)
    }
  }

  const handlePublishEvent = async () => {
    if (!event) return

    try {
      setIsSaving(true)
      setError("")

      const updatedEvent = await eventsApi.publish(event.id)
      setEvent(updatedEvent)
      setSuccessMessage("Event published successfully!")

      setTimeout(() => setSuccessMessage(""), 3000)
    } catch (err) {
      console.error("Failed to publish event:", err)
      setError("Failed to publish event. Make sure the event has at least one show with zones.")
    } finally {
      setIsSaving(false)
    }
  }

  const handleSaveShow = async (showId: string) => {
    try {
      setIsSaving(true)
      setError("")

      await showsApi.update(showId, showForm)
      setEditingShow(null)
      setShowForm({})
      await loadEventData()
      setSuccessMessage("Show updated successfully!")

      setTimeout(() => setSuccessMessage(""), 3000)
    } catch (err) {
      console.error("Failed to update show:", err)
      setError("Failed to update show")
    } finally {
      setIsSaving(false)
    }
  }

  const handleSaveZone = async (zoneId: string) => {
    try {
      setIsSaving(true)
      setError("")

      await zonesApi.update(zoneId, zoneForm)
      setEditingZone(null)
      setZoneForm({})
      await loadEventData()
      setSuccessMessage("Zone updated successfully!")

      setTimeout(() => setSuccessMessage(""), 3000)
    } catch (err) {
      console.error("Failed to update zone:", err)
      setError("Failed to update zone")
    } finally {
      setIsSaving(false)
    }
  }

  const handleToggleZoneActive = async (zone: ShowZoneResponse) => {
    try {
      setIsSaving(true)
      await zonesApi.update(zone.id, { is_active: !zone.is_active })
      await loadEventData()
      setSuccessMessage(`Zone ${zone.is_active ? "deactivated" : "activated"} successfully!`)
      setTimeout(() => setSuccessMessage(""), 3000)
    } catch (err) {
      console.error("Failed to toggle zone:", err)
      setError("Failed to toggle zone status")
    } finally {
      setIsSaving(false)
    }
  }

  const getStatusBadge = (status: string) => {
    const statusConfig: Record<string, { variant: "default" | "secondary" | "destructive" | "outline"; icon: React.ReactNode }> = {
      draft: { variant: "secondary", icon: <Edit className="h-3 w-3 mr-1" /> },
      published: { variant: "default", icon: <CheckCircle className="h-3 w-3 mr-1" /> },
      on_sale: { variant: "default", icon: <Ticket className="h-3 w-3 mr-1" /> },
      sold_out: { variant: "destructive", icon: <AlertCircle className="h-3 w-3 mr-1" /> },
      cancelled: { variant: "destructive", icon: <AlertCircle className="h-3 w-3 mr-1" /> },
      completed: { variant: "outline", icon: <CheckCircle className="h-3 w-3 mr-1" /> },
    }
    const config = statusConfig[status] || { variant: "secondary" as const, icon: null }
    return (
      <Badge variant={config.variant} className="flex items-center">
        {config.icon}
        {status.replace("_", " ").toUpperCase()}
      </Badge>
    )
  }

  const formatDate = (dateString: string) => {
    return new Date(dateString).toLocaleDateString("en-US", {
      year: "numeric",
      month: "short",
      day: "numeric",
    })
  }

  const formatTime = (dateString: string) => {
    return new Date(dateString).toLocaleTimeString("en-US", {
      hour: "2-digit",
      minute: "2-digit",
    })
  }

  // Calculate event stats from shows and zones
  const calculateStats = () => {
    let totalCapacity = 0
    let totalSold = 0
    let totalReserved = 0
    let totalRevenue = 0

    // Aggregate from all zones across all shows
    Object.values(zonesByShow).forEach((zones) => {
      zones.forEach((zone) => {
        totalCapacity += zone.total_seats
        totalSold += zone.sold_seats
        totalReserved += zone.reserved_seats
        totalRevenue += zone.sold_seats * zone.price
      })
    })

    const totalAvailable = totalCapacity - totalSold - totalReserved
    const occupancyRate = totalCapacity > 0 ? ((totalSold / totalCapacity) * 100).toFixed(1) : "0"

    return {
      totalCapacity,
      totalSold,
      totalReserved,
      totalAvailable,
      totalRevenue,
      occupancyRate,
    }
  }

  const stats = calculateStats()

  if (isLoading) {
    return (
      <div className="space-y-6 max-w-5xl">
        <div className="flex items-center gap-4">
          <div className="h-10 w-10 bg-muted animate-pulse rounded" />
          <div className="space-y-2">
            <div className="h-8 w-64 bg-muted animate-pulse rounded" />
            <div className="h-4 w-48 bg-muted animate-pulse rounded" />
          </div>
        </div>
        <div className="h-96 bg-muted animate-pulse rounded-lg" />
      </div>
    )
  }

  if (!event) {
    return (
      <div className="space-y-6 max-w-5xl">
        <div className="text-center py-12">
          <AlertCircle className="h-12 w-12 mx-auto text-muted-foreground mb-4" />
          <h2 className="text-xl font-semibold mb-2">Event Not Found</h2>
          <p className="text-muted-foreground mb-4">The event you are looking for does not exist.</p>
          <Link href="/organizer/events">
            <Button>Back to Events</Button>
          </Link>
        </div>
      </div>
    )
  }

  return (
    <div className="space-y-6 max-w-5xl">
      {/* Page Header */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-4">
          <Link href="/organizer/events">
            <Button variant="ghost" size="icon">
              <ArrowLeft className="h-4 w-4" />
            </Button>
          </Link>
          <div>
            <div className="flex items-center gap-3">
              <h1 className="text-3xl font-bold">{event.name}</h1>
              {getStatusBadge(event.status)}
            </div>
            <p className="text-muted-foreground mt-1">
              {event.city} &bull; Created {formatDate(event.created_at)}
            </p>
          </div>
        </div>
        <div className="flex items-center gap-2">
          <Link href={`/events/${event.id}`} target="_blank">
            <Button variant="outline">
              <Eye className="h-4 w-4 mr-2" />
              View Public
            </Button>
          </Link>
          {event.status === "draft" && (
            <Button onClick={handlePublishEvent} disabled={isSaving}>
              <CheckCircle className="h-4 w-4 mr-2" />
              Publish
            </Button>
          )}
        </div>
      </div>

      {/* Messages */}
      {error && (
        <div className="bg-destructive/10 text-destructive px-4 py-3 rounded-lg flex items-center gap-2">
          <AlertCircle className="h-4 w-4" />
          {error}
        </div>
      )}
      {successMessage && (
        <div className="bg-green-500/10 text-green-500 px-4 py-3 rounded-lg flex items-center gap-2">
          <CheckCircle className="h-4 w-4" />
          {successMessage}
        </div>
      )}

      {/* Stats Cards */}
      <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
        <Card>
          <CardContent className="p-4">
            <div className="flex items-center gap-3">
              <div className="p-2 rounded-lg bg-blue-500/10">
                <Users className="h-5 w-5 text-blue-500" />
              </div>
              <div>
                <p className="text-sm text-muted-foreground">Total Capacity</p>
                <p className="text-2xl font-bold">{stats.totalCapacity.toLocaleString()}</p>
              </div>
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardContent className="p-4">
            <div className="flex items-center gap-3">
              <div className="p-2 rounded-lg bg-green-500/10">
                <Ticket className="h-5 w-5 text-green-500" />
              </div>
              <div>
                <p className="text-sm text-muted-foreground">Tickets Sold</p>
                <p className="text-2xl font-bold">{stats.totalSold.toLocaleString()}</p>
              </div>
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardContent className="p-4">
            <div className="flex items-center gap-3">
              <div className="p-2 rounded-lg bg-yellow-500/10">
                <DollarSign className="h-5 w-5 text-yellow-500" />
              </div>
              <div>
                <p className="text-sm text-muted-foreground">Revenue</p>
                <p className="text-2xl font-bold">฿{stats.totalRevenue.toLocaleString()}</p>
              </div>
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardContent className="p-4">
            <div className="flex items-center gap-3">
              <div className="p-2 rounded-lg bg-purple-500/10">
                <TrendingUp className="h-5 w-5 text-purple-500" />
              </div>
              <div>
                <p className="text-sm text-muted-foreground">Occupancy</p>
                <p className="text-2xl font-bold">{stats.occupancyRate}%</p>
              </div>
            </div>
          </CardContent>
        </Card>
      </div>

      {/* Tabs */}
      <Tabs value={activeTab} onValueChange={setActiveTab}>
        <TabsList className="grid w-full grid-cols-3">
          <TabsTrigger value="details" className="flex items-center gap-2">
            <Settings className="h-4 w-4" />
            Details
          </TabsTrigger>
          <TabsTrigger value="shows" className="flex items-center gap-2">
            <Calendar className="h-4 w-4" />
            Shows ({shows.length})
          </TabsTrigger>
          <TabsTrigger value="zones" className="flex items-center gap-2">
            <Ticket className="h-4 w-4" />
            Zones
          </TabsTrigger>
        </TabsList>

        {/* Event Details Tab */}
        <TabsContent value="details" className="space-y-6">
          <Card>
            <CardHeader>
              <CardTitle>Basic Information</CardTitle>
              <CardDescription>Update your event details</CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="grid gap-4 md:grid-cols-2">
                <div className="space-y-2">
                  <Label htmlFor="name">Event Name</Label>
                  <Input
                    id="name"
                    value={editForm.name || ""}
                    onChange={(e) => setEditForm({ ...editForm, name: e.target.value })}
                  />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="city">City</Label>
                  <Input
                    id="city"
                    value={editForm.city || ""}
                    onChange={(e) => setEditForm({ ...editForm, city: e.target.value })}
                  />
                </div>
              </div>

              <div className="space-y-2">
                <Label htmlFor="short_description">Short Description</Label>
                <Input
                  id="short_description"
                  value={editForm.short_description || ""}
                  onChange={(e) => setEditForm({ ...editForm, short_description: e.target.value })}
                />
              </div>

              <div className="space-y-2">
                <Label htmlFor="description">Full Description</Label>
                <textarea
                  id="description"
                  value={editForm.description || ""}
                  onChange={(e) => setEditForm({ ...editForm, description: e.target.value })}
                  className="w-full min-h-[100px] px-3 py-2 border rounded-md bg-background"
                />
              </div>

              <div className="grid gap-4 md:grid-cols-2">
                <div className="space-y-2">
                  <Label htmlFor="venue_name">Venue Name</Label>
                  <Input
                    id="venue_name"
                    value={editForm.venue_name || ""}
                    onChange={(e) => setEditForm({ ...editForm, venue_name: e.target.value })}
                  />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="venue_address">Venue Address</Label>
                  <Input
                    id="venue_address"
                    value={editForm.venue_address || ""}
                    onChange={(e) => setEditForm({ ...editForm, venue_address: e.target.value })}
                  />
                </div>
              </div>

              <div className="grid gap-4 md:grid-cols-2">
                <div className="space-y-2">
                  <Label htmlFor="poster_url">Poster Image URL</Label>
                  <Input
                    id="poster_url"
                    value={editForm.poster_url || ""}
                    onChange={(e) => setEditForm({ ...editForm, poster_url: e.target.value })}
                  />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="max_tickets_per_user">Max Tickets Per User</Label>
                  <Input
                    id="max_tickets_per_user"
                    type="number"
                    min={1}
                    value={editForm.max_tickets_per_user || 4}
                    onChange={(e) => setEditForm({ ...editForm, max_tickets_per_user: parseInt(e.target.value) || 1 })}
                  />
                </div>
              </div>

              <div className="flex justify-end">
                <Button onClick={handleSaveEvent} disabled={isSaving}>
                  <Save className="h-4 w-4 mr-2" />
                  {isSaving ? "Saving..." : "Save Changes"}
                </Button>
              </div>
            </CardContent>
          </Card>
        </TabsContent>

        {/* Shows Tab */}
        <TabsContent value="shows" className="space-y-4">
          {shows.length === 0 ? (
            <Card>
              <CardContent className="flex flex-col items-center justify-center py-12">
                <Calendar className="h-12 w-12 text-muted-foreground mb-4" />
                <h3 className="font-medium mb-2">No shows yet</h3>
                <p className="text-sm text-muted-foreground">This event doesn&apos;t have any shows.</p>
              </CardContent>
            </Card>
          ) : (
            shows.map((show) => (
              <Card key={show.id}>
                <CardHeader className="flex flex-row items-center justify-between">
                  <div>
                    <CardTitle className="text-lg">{show.name}</CardTitle>
                    <CardDescription>
                      {formatDate(show.show_date)} &bull; {formatTime(show.start_time)} - {formatTime(show.end_time)}
                    </CardDescription>
                  </div>
                  <div className="flex items-center gap-2">
                    {getStatusBadge(show.status)}
                    {editingShow === show.id ? (
                      <Button size="sm" onClick={() => handleSaveShow(show.id)} disabled={isSaving}>
                        <Save className="h-4 w-4 mr-1" />
                        Save
                      </Button>
                    ) : (
                      <Button
                        size="sm"
                        variant="outline"
                        onClick={() => {
                          setEditingShow(show.id)
                          setShowForm({
                            name: show.name,
                            status: show.status,
                          })
                        }}
                      >
                        <Edit className="h-4 w-4 mr-1" />
                        Edit
                      </Button>
                    )}
                  </div>
                </CardHeader>
                {editingShow === show.id && (
                  <CardContent className="space-y-4 border-t pt-4">
                    <div className="grid gap-4 md:grid-cols-2">
                      <div className="space-y-2">
                        <Label>Show Name</Label>
                        <Input
                          value={showForm.name || ""}
                          onChange={(e) => setShowForm({ ...showForm, name: e.target.value })}
                        />
                      </div>
                      <div className="space-y-2">
                        <Label>Status</Label>
                        <select
                          value={showForm.status || show.status}
                          onChange={(e) => setShowForm({ ...showForm, status: e.target.value })}
                          className="w-full h-10 px-3 border rounded-md bg-background"
                        >
                          <option value="scheduled">Scheduled</option>
                          <option value="on_sale">On Sale</option>
                          <option value="sold_out">Sold Out</option>
                          <option value="cancelled">Cancelled</option>
                          <option value="completed">Completed</option>
                        </select>
                      </div>
                    </div>
                    <div className="flex justify-end">
                      <Button variant="ghost" onClick={() => setEditingShow(null)}>
                        Cancel
                      </Button>
                    </div>
                  </CardContent>
                )}
                <CardContent className="pt-0">
                  <div className="grid grid-cols-3 gap-4 text-sm">
                    <div>
                      <span className="text-muted-foreground">Capacity:</span>{" "}
                      <span className="font-medium">{show.total_capacity}</span>
                    </div>
                    <div>
                      <span className="text-muted-foreground">Reserved:</span>{" "}
                      <span className="font-medium">{show.reserved_count}</span>
                    </div>
                    <div>
                      <span className="text-muted-foreground">Sold:</span>{" "}
                      <span className="font-medium">{show.sold_count}</span>
                    </div>
                  </div>
                </CardContent>
              </Card>
            ))
          )}
        </TabsContent>

        {/* Zones Tab */}
        <TabsContent value="zones" className="space-y-4">
          {shows.length === 0 ? (
            <Card>
              <CardContent className="flex flex-col items-center justify-center py-12">
                <Ticket className="h-12 w-12 text-muted-foreground mb-4" />
                <h3 className="font-medium mb-2">No zones available</h3>
                <p className="text-sm text-muted-foreground">Create shows first to manage zones.</p>
              </CardContent>
            </Card>
          ) : (
            shows.map((show) => (
              <Card key={show.id}>
                <CardHeader>
                  <CardTitle className="text-lg">{show.name} - Zones</CardTitle>
                  <CardDescription>{formatDate(show.show_date)}</CardDescription>
                </CardHeader>
                <CardContent>
                  {!zonesByShow[show.id] || zonesByShow[show.id].length === 0 ? (
                    <p className="text-sm text-muted-foreground py-4 text-center">No zones for this show</p>
                  ) : (
                    <div className="space-y-3">
                      {zonesByShow[show.id].map((zone) => (
                        <div
                          key={zone.id}
                          className={`p-4 rounded-lg border ${zone.is_active ? "bg-background" : "bg-muted/50 opacity-60"}`}
                        >
                          {editingZone === zone.id ? (
                            <div className="space-y-4">
                              <div className="grid gap-4 md:grid-cols-4">
                                <div className="space-y-1">
                                  <Label className="text-xs">Zone Name</Label>
                                  <Input
                                    value={zoneForm.name || ""}
                                    onChange={(e) => setZoneForm({ ...zoneForm, name: e.target.value })}
                                    className="h-9"
                                  />
                                </div>
                                <div className="space-y-1">
                                  <Label className="text-xs">Price (THB)</Label>
                                  <Input
                                    type="number"
                                    value={zoneForm.price || 0}
                                    onChange={(e) => setZoneForm({ ...zoneForm, price: parseInt(e.target.value) || 0 })}
                                    className="h-9"
                                  />
                                </div>
                                <div className="space-y-1">
                                  <Label className="text-xs">Total Seats</Label>
                                  <Input
                                    type="number"
                                    value={zoneForm.total_seats || 0}
                                    onChange={(e) => setZoneForm({ ...zoneForm, total_seats: parseInt(e.target.value) || 0 })}
                                    className="h-9"
                                  />
                                </div>
                                <div className="space-y-1">
                                  <Label className="text-xs">Max Per Order</Label>
                                  <Input
                                    type="number"
                                    value={zoneForm.max_per_order || 0}
                                    onChange={(e) => setZoneForm({ ...zoneForm, max_per_order: parseInt(e.target.value) || 0 })}
                                    className="h-9"
                                  />
                                </div>
                              </div>
                              <div className="flex justify-end gap-2">
                                <Button variant="ghost" size="sm" onClick={() => setEditingZone(null)}>
                                  Cancel
                                </Button>
                                <Button size="sm" onClick={() => handleSaveZone(zone.id)} disabled={isSaving}>
                                  <Save className="h-4 w-4 mr-1" />
                                  Save
                                </Button>
                              </div>
                            </div>
                          ) : (
                            <div className="flex items-center justify-between">
                              <div className="flex items-center gap-4">
                                <div
                                  className="w-4 h-4 rounded-full"
                                  style={{ backgroundColor: zone.color || "#888" }}
                                />
                                <div>
                                  <div className="font-medium">
                                    {zone.name}
                                  </div>
                                  <div className="text-sm text-muted-foreground">
                                    {zone.price.toLocaleString()} THB &bull; {zone.available_seats}/{zone.total_seats} available
                                  </div>
                                </div>
                              </div>
                              <div className="flex items-center gap-4">
                                {/* Active Toggle Switch */}
                                <div className="flex items-center gap-2">
                                  <Switch
                                    checked={zone.is_active}
                                    onCheckedChange={() => handleToggleZoneActive(zone)}
                                    disabled={isSaving}
                                  />
                                  <span className={`text-sm font-medium ${zone.is_active ? "text-green-500" : "text-muted-foreground"}`}>
                                    {zone.is_active ? "Active" : "Inactive"}
                                  </span>
                                </div>
                                {/* Edit Button */}
                                <Button
                                  variant="ghost"
                                  size="sm"
                                  onClick={() => {
                                    setEditingZone(zone.id)
                                    setZoneForm({
                                      name: zone.name,
                                      price: zone.price,
                                      total_seats: zone.total_seats,
                                      max_per_order: zone.max_per_order,
                                      description: zone.description,
                                    })
                                  }}
                                >
                                  <Edit className="h-4 w-4" />
                                </Button>
                              </div>
                            </div>
                          )}
                        </div>
                      ))}
                    </div>
                  )}
                </CardContent>
              </Card>
            ))
          )}
        </TabsContent>
      </Tabs>
    </div>
  )
}
