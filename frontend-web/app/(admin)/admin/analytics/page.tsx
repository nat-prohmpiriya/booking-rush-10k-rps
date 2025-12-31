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
  PieChart,
  Pie,
  Cell,
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

// Overview Stats Cards
function OverviewCards() {
  const { data, isLoading } = useDashboardOverview()

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

  const stats = [
    {
      title: 'Total Revenue',
      value: formatCurrency(data?.total_revenue || 0),
      change: data?.revenue_change_percent || 0,
      icon: DollarSign,
      color: 'text-green-500',
      bgColor: 'bg-green-500/10',
    },
    {
      title: 'Total Bookings',
      value: formatNumber(data?.total_bookings || 0),
      change: data?.bookings_change_percent || 0,
      icon: Ticket,
      color: 'text-blue-500',
      bgColor: 'bg-blue-500/10',
    },
    {
      title: 'Tickets Sold',
      value: formatNumber(data?.total_tickets_sold || 0),
      change: null,
      icon: BarChart3,
      color: 'text-purple-500',
      bgColor: 'bg-purple-500/10',
    },
    {
      title: 'Active Events',
      value: formatNumber(data?.active_events || 0),
      change: null,
      icon: Calendar,
      color: 'text-amber-500',
      bgColor: 'bg-amber-500/10',
    },
  ]

  return (
    <div data-testid="admin-analytics-stats" className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
      {stats.map((stat) => (
        <Card key={stat.title} data-testid={`admin-analytics-stat-${stat.title.toLowerCase().replace(/\s+/g, '-')}`}>
          <CardHeader className="flex flex-row items-center justify-between pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground">
              {stat.title}
            </CardTitle>
            <div className={`p-2 rounded-lg ${stat.bgColor}`}>
              <stat.icon className={`h-4 w-4 ${stat.color}`} />
            </div>
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

// Revenue Chart
function RevenueChart() {
  const [period, setPeriod] = useState<SalesReportFilter['period']>('day')
  const { data, isLoading } = useSalesReport({ period })

  if (isLoading) {
    return (
      <Card className="col-span-3">
        <CardHeader>
          <Skeleton className="h-6 w-32" />
        </CardHeader>
        <CardContent>
          <Skeleton className="h-[350px] w-full" />
        </CardContent>
      </Card>
    )
  }

  const chartData = data?.data.map((item) => ({
    ...item,
    date: formatDate(item.period),
  })) || []

  return (
    <Card data-testid="admin-analytics-revenue-chart" className="col-span-3">
      <CardHeader className="flex flex-row items-center justify-between">
        <div>
          <CardTitle>Revenue Trend</CardTitle>
          <CardDescription>
            {data?.start_date} to {data?.end_date}
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
        <div className="h-[350px]">
          <ResponsiveContainer width="100%" height="100%">
            <AreaChart data={chartData}>
              <defs>
                <linearGradient id="colorRevenueAdmin" x1="0" y1="0" x2="0" y2="1">
                  <stop offset="5%" stopColor="#ef4444" stopOpacity={0.3} />
                  <stop offset="95%" stopColor="#ef4444" stopOpacity={0} />
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
                    const d = payload[0].payload
                    return (
                      <div className="rounded-lg border bg-background p-2 shadow-sm">
                        <div className="text-sm font-medium">{d.date}</div>
                        <div className="text-xs text-muted-foreground">
                          Revenue: {formatCurrency(d.revenue)}
                        </div>
                        <div className="text-xs text-muted-foreground">
                          Bookings: {d.bookings}
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
                stroke="#ef4444"
                fillOpacity={1}
                fill="url(#colorRevenueAdmin)"
              />
            </AreaChart>
          </ResponsiveContainer>
        </div>
      </CardContent>
    </Card>
  )
}

// Top Events Bar Chart
function TopEventsChart() {
  const { data, isLoading } = useTopEvents(5)

  const COLORS = ['#ef4444', '#f97316', '#eab308', '#22c55e', '#3b82f6']

  if (isLoading) {
    return (
      <Card>
        <CardHeader>
          <Skeleton className="h-6 w-32" />
        </CardHeader>
        <CardContent>
          <Skeleton className="h-[350px] w-full" />
        </CardContent>
      </Card>
    )
  }

  const chartData = data?.map((event) => ({
    name: event.event_name.length > 15
      ? event.event_name.substring(0, 15) + '...'
      : event.event_name,
    revenue: event.total_revenue,
    bookings: event.total_bookings,
  })) || []

  return (
    <Card data-testid="admin-analytics-top-events-chart">
      <CardHeader>
        <CardTitle>Top Events</CardTitle>
        <CardDescription>By revenue</CardDescription>
      </CardHeader>
      <CardContent>
        <div className="h-[350px]">
          <ResponsiveContainer width="100%" height="100%">
            <BarChart data={chartData} layout="vertical">
              <CartesianGrid strokeDasharray="3 3" className="stroke-muted" />
              <XAxis
                type="number"
                className="text-xs"
                tick={{ fill: 'hsl(var(--muted-foreground))' }}
                tickFormatter={(value) => `${(value / 1000).toFixed(0)}k`}
              />
              <YAxis
                type="category"
                dataKey="name"
                className="text-xs"
                tick={{ fill: 'hsl(var(--muted-foreground))' }}
                width={100}
              />
              <Tooltip
                content={({ active, payload }) => {
                  if (active && payload && payload.length) {
                    const d = payload[0].payload
                    return (
                      <div className="rounded-lg border bg-background p-2 shadow-sm">
                        <div className="text-sm font-medium">{d.name}</div>
                        <div className="text-xs text-muted-foreground">
                          Revenue: {formatCurrency(d.revenue)}
                        </div>
                        <div className="text-xs text-muted-foreground">
                          Bookings: {d.bookings}
                        </div>
                      </div>
                    )
                  }
                  return null
                }}
              />
              <Bar dataKey="revenue" fill="#ef4444" radius={[0, 4, 4, 0]} />
            </BarChart>
          </ResponsiveContainer>
        </div>
      </CardContent>
    </Card>
  )
}

// Summary Stats
function SummaryStats() {
  const { data } = useSalesReport({ period: 'month' })

  const summaryData = [
    {
      label: 'Total Revenue',
      value: formatCurrency(data?.total_revenue || 0),
    },
    {
      label: 'Total Bookings',
      value: formatNumber(data?.total_bookings || 0),
    },
    {
      label: 'Tickets Sold',
      value: formatNumber(data?.total_tickets_sold || 0),
    },
    {
      label: 'Avg. per Booking',
      value: data?.total_bookings
        ? formatCurrency((data.total_revenue || 0) / data.total_bookings)
        : formatCurrency(0),
    },
  ]

  return (
    <Card data-testid="admin-analytics-summary">
      <CardHeader>
        <CardTitle>Period Summary</CardTitle>
        <CardDescription>
          {data?.start_date} - {data?.end_date}
        </CardDescription>
      </CardHeader>
      <CardContent>
        <div className="space-y-4">
          {summaryData.map((item) => (
            <div key={item.label} className="flex justify-between items-center">
              <span className="text-sm text-muted-foreground">{item.label}</span>
              <span className="font-medium">{item.value}</span>
            </div>
          ))}
        </div>
      </CardContent>
    </Card>
  )
}

// Main Admin Analytics Page
export default function AdminAnalyticsPage() {
  return (
    <div data-testid="admin-analytics-page" className="space-y-8">
      <div>
        <h1 data-testid="admin-analytics-title" className="text-3xl font-bold">System Analytics</h1>
        <p className="text-muted-foreground">
          Platform-wide performance metrics
        </p>
      </div>

      {/* Overview Stats */}
      <OverviewCards />

      {/* Charts */}
      <div data-testid="admin-analytics-charts" className="grid gap-4 lg:grid-cols-4">
        <RevenueChart />
        <SummaryStats />
      </div>

      {/* Top Events */}
      <TopEventsChart />
    </div>
  )
}
