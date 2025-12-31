'use client'

import { useState } from 'react'
import { useAllEvents } from '@/hooks/use-admin'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { Badge } from '@/components/ui/badge'
import { ChevronLeft, ChevronRight, Calendar, MapPin, ExternalLink } from 'lucide-react'
import Link from 'next/link'
import Image from 'next/image'

function formatDate(dateString: string): string {
  return new Date(dateString).toLocaleDateString('th-TH', {
    day: 'numeric',
    month: 'short',
    year: 'numeric',
  })
}

function formatCurrency(amount: number): string {
  return new Intl.NumberFormat('th-TH', {
    style: 'currency',
    currency: 'THB',
    minimumFractionDigits: 0,
  }).format(amount)
}

const statusColors: Record<string, string> = {
  published: 'bg-green-500',
  draft: 'bg-gray-500',
  cancelled: 'bg-red-500',
  completed: 'bg-blue-500',
}

const saleStatusColors: Record<string, string> = {
  on_sale: 'bg-green-500',
  scheduled: 'bg-yellow-500',
  sold_out: 'bg-red-500',
  cancelled: 'bg-gray-500',
  completed: 'bg-blue-500',
}

export default function AdminEventsPage() {
  const [status, setStatus] = useState<string>('')
  const [page, setPage] = useState(1)
  const limit = 10

  const { data, isLoading, error } = useAllEvents({
    status: status || undefined,
    page,
    limit,
  })

  return (
    <div data-testid="admin-events-page" className="space-y-6">
      <div>
        <h1 data-testid="admin-events-title" className="text-3xl font-bold">All Events</h1>
        <p className="text-muted-foreground">
          View all events across all organizers
        </p>
      </div>

      {/* Filters */}
      <Card data-testid="admin-events-filters">
        <CardContent className="pt-6">
          <div className="flex gap-4">
            <Select value={status} onValueChange={(v) => { setStatus(v); setPage(1); }}>
              <SelectTrigger data-testid="admin-events-filter" className="w-40">
                <SelectValue placeholder="All status" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="">All status</SelectItem>
                <SelectItem value="published">Published</SelectItem>
                <SelectItem value="draft">Draft</SelectItem>
                <SelectItem value="cancelled">Cancelled</SelectItem>
                <SelectItem value="completed">Completed</SelectItem>
              </SelectContent>
            </Select>
          </div>
        </CardContent>
      </Card>

      {/* Events Table */}
      <Card data-testid="admin-events-table-card">
        <CardHeader>
          <CardTitle>Events</CardTitle>
          <CardDescription>
            {data?.meta?.total || 0} events found
          </CardDescription>
        </CardHeader>
        <CardContent>
          {isLoading ? (
            <div className="space-y-4">
              {[...Array(5)].map((_, i) => (
                <div key={i} className="flex items-center gap-4">
                  <Skeleton className="h-16 w-24 rounded" />
                  <div className="flex-1">
                    <Skeleton className="h-4 w-48 mb-2" />
                    <Skeleton className="h-3 w-32" />
                  </div>
                  <Skeleton className="h-6 w-20" />
                </div>
              ))}
            </div>
          ) : error ? (
            <div className="text-center text-muted-foreground py-8">
              {error}
            </div>
          ) : !data?.data || data.data.length === 0 ? (
            <div className="text-center text-muted-foreground py-8">
              No events found
            </div>
          ) : (
            <>
              <Table data-testid="admin-events-table">
                <TableHeader>
                  <TableRow>
                    <TableHead>Event</TableHead>
                    <TableHead>Venue</TableHead>
                    <TableHead>Status</TableHead>
                    <TableHead>Sale Status</TableHead>
                    <TableHead>Price</TableHead>
                    <TableHead>Created</TableHead>
                    <TableHead className="text-right">Actions</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {data.data.map((event) => (
                    <TableRow key={event.id}>
                      <TableCell>
                        <div className="flex items-center gap-3">
                          {event.poster_url ? (
                            <Image
                              src={event.poster_url}
                              alt={event.name}
                              width={60}
                              height={40}
                              className="rounded object-cover"
                            />
                          ) : (
                            <div className="w-[60px] h-[40px] rounded bg-muted flex items-center justify-center">
                              <Calendar className="h-4 w-4 text-muted-foreground" />
                            </div>
                          )}
                          <div>
                            <p className="font-medium truncate max-w-[200px]">
                              {event.name}
                            </p>
                            <p className="text-xs text-muted-foreground">
                              {event.slug}
                            </p>
                          </div>
                        </div>
                      </TableCell>
                      <TableCell>
                        <div className="flex items-center gap-1 text-sm text-muted-foreground">
                          <MapPin className="h-3 w-3" />
                          <span className="truncate max-w-[150px]">
                            {event.venue_name || 'TBA'}
                          </span>
                        </div>
                      </TableCell>
                      <TableCell>
                        <Badge className={`${statusColors[event.status] || 'bg-gray-500'} text-white`}>
                          {event.status}
                        </Badge>
                      </TableCell>
                      <TableCell>
                        <Badge variant="outline" className={`${saleStatusColors[event.sale_status] || 'bg-gray-500'} text-white border-0`}>
                          {event.sale_status?.replace('_', ' ') || 'N/A'}
                        </Badge>
                      </TableCell>
                      <TableCell>
                        {formatCurrency(event.min_price || 0)}
                      </TableCell>
                      <TableCell className="text-muted-foreground text-sm">
                        {formatDate(event.created_at)}
                      </TableCell>
                      <TableCell className="text-right">
                        <Link href={`/events/${event.id}`} target="_blank">
                          <Button variant="ghost" size="sm">
                            <ExternalLink className="h-4 w-4" />
                          </Button>
                        </Link>
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>

              {/* Pagination */}
              {data.meta && data.meta.total_pages > 1 && (
                <div data-testid="admin-events-pagination" className="flex items-center justify-between mt-4">
                  <p className="text-sm text-muted-foreground">
                    Page {data.meta.page} of {data.meta.total_pages}
                  </p>
                  <div className="flex gap-2">
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={() => setPage((p) => Math.max(1, p - 1))}
                      disabled={page <= 1}
                    >
                      <ChevronLeft className="h-4 w-4" />
                    </Button>
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={() => setPage((p) => Math.min(data.meta.total_pages, p + 1))}
                      disabled={page >= data.meta.total_pages}
                    >
                      <ChevronRight className="h-4 w-4" />
                    </Button>
                  </div>
                </div>
              )}
            </>
          )}
        </CardContent>
      </Card>
    </div>
  )
}
