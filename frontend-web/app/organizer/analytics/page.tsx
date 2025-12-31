'use client'

import { useState } from 'react'
import {
  useDashboardOverview,
  useSalesReport,
  useTopEvents,
  useRecentBookings,
} from '@/hooks/use-analytics'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import {
  DollarSign,
  Ticket,
  Calendar,
  TrendingUp,
  TrendingDown,
  BarChart3,
} from 'lucide-react'
import {
  AreaChart,
  Area,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
  BarChart,
  Bar,
} from 'recharts'
import type { SalesReportFilter } from '@/lib/api/types'

function formatCurrency(amount: number): string {
  return new Intl.NumberFormat('th-TH', {
    style: 'currency',
    currency: 'THB',
    minimumFractionDigits: 0,
  }).format(amount)
}

function formatNumber(num: number): string {
  return new Intl.NumberFormat('th-TH').format(num)
}

function formatDate(dateString: string): string {
  return new Date(dateString).toLocaleDateString('th-TH', {
    day: 'numeric',
    month: 'short',
  })
}

function formatDateTime(dateString: string): string {
  return new Date(dateString).toLocaleString('th-TH', {
    day: 'numeric',
    month: 'short',
    hour: '2-digit',
    minute: '2-digit',
  })
}

// Overview Stats Cards
function OverviewCards() {
  const { data, isLoading, error } = useDashboardOverview()

  if (isLoading) {
    return (
      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
        {[...Array(4)].map((_, i) => (
          <Card key={i}>
            <CardHeader className="flex flex-row items-center justify-between pb-2">
              <Skeleton className="h-4 w-24" />
              <Skeleton className="h-4 w-4" />
            </CardHeader>
            <CardContent>
              <Skeleton className="h-8 w-32 mb-1" />
              <Skeleton className="h-3 w-20" />
            </CardContent>
          </Card>
        ))}
      </div>
    )
  }

  if (error || !data) {
    return (
      <div className="text-center text-muted-foreground py-8">
        Failed to load dashboard data
      </div>
    )
  }

  const stats = [
    {
      title: 'Total Revenue',
      value: formatCurrency(data.total_revenue),
      change: data.revenue_change_percent,
      icon: DollarSign,
    },
    {
      title: 'Total Bookings',
      value: formatNumber(data.total_bookings),
      change: data.bookings_change_percent,
      icon: Ticket,
    },
    {
      title: 'Tickets Sold',
      value: formatNumber(data.total_tickets_sold),
      change: null,
      icon: BarChart3,
    },
    {
      title: 'Active Events',
      value: formatNumber(data.active_events),
      change: null,
      icon: Calendar,
    },
  ]

  return (
    <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
      {stats.map((stat) => (
        <Card key={stat.title}>
          <CardHeader className="flex flex-row items-center justify-between pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground">
              {stat.title}
            </CardTitle>
            <stat.icon className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{stat.value}</div>
            {stat.change !== null && (
              <p className={`text-xs flex items-center gap-1 ${
                stat.change >= 0 ? 'text-green-500' : 'text-red-500'
              }`}>
                {stat.change >= 0 ? (
                  <TrendingUp className="h-3 w-3" />
                ) : (
                  <TrendingDown className="h-3 w-3" />
                )}
                {Math.abs(stat.change).toFixed(1)}% vs last month
              </p>
            )}
          </CardContent>
        </Card>
      ))}
    </div>
  )
}

// Sales Chart
function SalesChart() {
  const [period, setPeriod] = useState<SalesReportFilter['period']>('day')
  const { data, isLoading, error } = useSalesReport({ period })

  if (isLoading) {
    return (
      <Card className="col-span-4">
        <CardHeader>
          <Skeleton className="h-6 w-32" />
        </CardHeader>
        <CardContent>
          <Skeleton className="h-[300px] w-full" />
        </CardContent>
      </Card>
    )
  }

  if (error || !data) {
    return (
      <Card className="col-span-4">
        <CardHeader>
          <CardTitle>Sales Overview</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="text-center text-muted-foreground py-8">
            Failed to load sales data
          </div>
        </CardContent>
      </Card>
    )
  }

  const chartData = data.data.map((item) => ({
    ...item,
    date: formatDate(item.period),
  }))

  return (
    <Card className="col-span-4">
      <CardHeader className="flex flex-row items-center justify-between">
        <div>
          <CardTitle>Sales Overview</CardTitle>
          <CardDescription>
            {data.start_date} - {data.end_date}
          </CardDescription>
        </div>
        <Select value={period} onValueChange={(v) => setPeriod(v as SalesReportFilter['period'])}>
          <SelectTrigger className="w-32">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="day">Daily</SelectItem>
            <SelectItem value="week">Weekly</SelectItem>
            <SelectItem value="month">Monthly</SelectItem>
          </SelectContent>
        </Select>
      </CardHeader>
      <CardContent>
        <div className="h-[300px]">
          <ResponsiveContainer width="100%" height="100%">
            <AreaChart data={chartData}>
              <defs>
                <linearGradient id="colorRevenue" x1="0" y1="0" x2="0" y2="1">
                  <stop offset="5%" stopColor="hsl(var(--primary))" stopOpacity={0.3} />
                  <stop offset="95%" stopColor="hsl(var(--primary))" stopOpacity={0} />
                </linearGradient>
              </defs>
              <CartesianGrid strokeDasharray="3 3" className="stroke-muted" />
              <XAxis
                dataKey="date"
                className="text-xs"
                tick={{ fill: 'hsl(var(--muted-foreground))' }}
              />
              <YAxis
                className="text-xs"
                tick={{ fill: 'hsl(var(--muted-foreground))' }}
                tickFormatter={(value) => `${(value / 1000).toFixed(0)}k`}
              />
              <Tooltip
                content={({ active, payload }) => {
                  if (active && payload && payload.length) {
                    const data = payload[0].payload
                    return (
                      <div className="rounded-lg border bg-background p-2 shadow-sm">
                        <div className="text-sm font-medium">{data.date}</div>
                        <div className="text-xs text-muted-foreground">
                          Revenue: {formatCurrency(data.revenue)}
                        </div>
                        <div className="text-xs text-muted-foreground">
                          Bookings: {data.bookings}
                        </div>
                        <div className="text-xs text-muted-foreground">
                          Tickets: {data.tickets_sold}
                        </div>
                      </div>
                    )
                  }
                  return null
                }}
              />
              <Area
                type="monotone"
                dataKey="revenue"
                stroke="hsl(var(--primary))"
                fillOpacity={1}
                fill="url(#colorRevenue)"
              />
            </AreaChart>
          </ResponsiveContainer>
        </div>
      </CardContent>
    </Card>
  )
}

// Top Events
function TopEvents() {
  const { data, isLoading, error } = useTopEvents(5)

  if (isLoading) {
    return (
      <Card className="col-span-2">
        <CardHeader>
          <Skeleton className="h-6 w-32" />
        </CardHeader>
        <CardContent>
          {[...Array(5)].map((_, i) => (
            <div key={i} className="flex items-center gap-4 py-3">
              <Skeleton className="h-4 w-4" />
              <Skeleton className="h-4 flex-1" />
              <Skeleton className="h-4 w-20" />
            </div>
          ))}
        </CardContent>
      </Card>
    )
  }

  if (error || !data) {
    return (
      <Card className="col-span-2">
        <CardHeader>
          <CardTitle>Top Events</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="text-center text-muted-foreground py-4">
            Failed to load top events
          </div>
        </CardContent>
      </Card>
    )
  }

  const maxRevenue = Math.max(...data.map((e) => e.total_revenue), 1)

  return (
    <Card className="col-span-2">
      <CardHeader>
        <CardTitle>Top Events</CardTitle>
        <CardDescription>By revenue</CardDescription>
      </CardHeader>
      <CardContent>
        {data.length === 0 ? (
          <div className="text-center text-muted-foreground py-4">
            No events yet
          </div>
        ) : (
          <div className="space-y-4">
            {data.map((event, index) => (
              <div key={event.event_id} className="space-y-2">
                <div className="flex items-center justify-between text-sm">
                  <div className="flex items-center gap-2">
                    <span className="text-muted-foreground w-4">{index + 1}.</span>
                    <span className="font-medium truncate max-w-[200px]">
                      {event.event_name}
                    </span>
                  </div>
                  <span className="font-medium">
                    {formatCurrency(event.total_revenue)}
                  </span>
                </div>
                <div className="h-2 rounded-full bg-muted overflow-hidden">
                  <div
                    className="h-full bg-primary rounded-full transition-all"
                    style={{ width: `${(event.total_revenue / maxRevenue) * 100}%` }}
                  />
                </div>
                <div className="flex justify-between text-xs text-muted-foreground">
                  <span>{event.total_bookings} bookings</span>
                  <span>{event.total_tickets} tickets</span>
                </div>
              </div>
            ))}
          </div>
        )}
      </CardContent>
    </Card>
  )
}

// Recent Bookings
function RecentBookings() {
  const { data, isLoading, error } = useRecentBookings(10)

  if (isLoading) {
    return (
      <Card className="col-span-2">
        <CardHeader>
          <Skeleton className="h-6 w-32" />
        </CardHeader>
        <CardContent>
          {[...Array(5)].map((_, i) => (
            <div key={i} className="flex items-center gap-4 py-3 border-b last:border-0">
              <Skeleton className="h-8 w-8 rounded-full" />
              <div className="flex-1">
                <Skeleton className="h-4 w-32 mb-1" />
                <Skeleton className="h-3 w-24" />
              </div>
              <Skeleton className="h-4 w-16" />
            </div>
          ))}
        </CardContent>
      </Card>
    )
  }

  if (error || !data) {
    return (
      <Card className="col-span-2">
        <CardHeader>
          <CardTitle>Recent Bookings</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="text-center text-muted-foreground py-4">
            Failed to load recent bookings
          </div>
        </CardContent>
      </Card>
    )
  }

  const statusColors: Record<string, string> = {
    confirmed: 'bg-green-500',
    completed: 'bg-blue-500',
    pending: 'bg-yellow-500',
    reserved: 'bg-orange-500',
    cancelled: 'bg-red-500',
    expired: 'bg-gray-500',
  }

  return (
    <Card className="col-span-2">
      <CardHeader>
        <CardTitle>Recent Bookings</CardTitle>
        <CardDescription>Latest 10 bookings</CardDescription>
      </CardHeader>
      <CardContent>
        {data.length === 0 ? (
          <div className="text-center text-muted-foreground py-4">
            No bookings yet
          </div>
        ) : (
          <div className="space-y-1">
            {data.map((booking) => (
              <div
                key={booking.booking_id}
                className="flex items-center gap-3 py-2 border-b last:border-0"
              >
                <div
                  className={`h-2 w-2 rounded-full ${
                    statusColors[booking.status] || 'bg-gray-500'
                  }`}
                />
                <div className="flex-1 min-w-0">
                  <div className="text-sm font-medium truncate">
                    {booking.event_name}
                  </div>
                  <div className="text-xs text-muted-foreground">
                    {booking.zone_name} x{booking.quantity} &bull;{' '}
                    {formatDateTime(booking.created_at)}
                  </div>
                </div>
                <div className="text-sm font-medium">
                  {formatCurrency(booking.total_price)}
                </div>
              </div>
            ))}
          </div>
        )}
      </CardContent>
    </Card>
  )
}

// Main Analytics Page
export default function AnalyticsPage() {
  return (
    <div className="space-y-8" data-testid="organizer-analytics-page">
      <div>
        <h1 className="text-3xl font-bold">Analytics</h1>
        <p className="text-muted-foreground">
          Track your events performance and revenue
        </p>
      </div>

      {/* Overview Stats */}
      <div data-testid="organizer-analytics-stats">
        <OverviewCards />
      </div>

      {/* Charts & Lists */}
      <div className="grid gap-4 lg:grid-cols-4" data-testid="organizer-analytics-charts">
        {/* Sales Chart - Full Width */}
        <SalesChart />

        {/* Top Events */}
        <TopEvents />

        {/* Recent Bookings */}
        <RecentBookings />
      </div>
    </div>
  )
}
