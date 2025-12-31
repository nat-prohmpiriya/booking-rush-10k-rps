'use client'

import { useSystemStats } from '@/hooks/use-admin'
import { useDashboardOverview, useRecentBookings } from '@/hooks/use-analytics'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'
import {
  Users,
  Calendar,
  Ticket,
  DollarSign,
  TrendingUp,
  TrendingDown,
  Building2,
  Activity,
} from 'lucide-react'
import Link from 'next/link'

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

function formatDateTime(dateString: string): string {
  return new Date(dateString).toLocaleString('th-TH', {
    day: 'numeric',
    month: 'short',
    hour: '2-digit',
    minute: '2-digit',
  })
}

// System Stats Cards
function SystemStatsCards() {
  const { data: systemStats, isLoading: systemLoading } = useSystemStats()
  const { data: analytics, isLoading: analyticsLoading } = useDashboardOverview()

  const isLoading = systemLoading || analyticsLoading

  if (isLoading) {
    return (
      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
        {[...Array(8)].map((_, i) => (
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
      title: 'Total Users',
      value: formatNumber(systemStats?.total_users || 0),
      subValue: `+${formatNumber(systemStats?.users_this_month || 0)} this month`,
      icon: Users,
      color: 'text-blue-500',
      bgColor: 'bg-blue-500/10',
    },
    {
      title: 'Total Events',
      value: formatNumber(systemStats?.total_events || 0),
      subValue: `${formatNumber(analytics?.active_events || 0)} active`,
      icon: Calendar,
      color: 'text-purple-500',
      bgColor: 'bg-purple-500/10',
    },
    {
      title: 'Total Bookings',
      value: formatNumber(analytics?.total_bookings || systemStats?.total_bookings || 0),
      subValue: analytics?.bookings_change_percent
        ? `${analytics.bookings_change_percent >= 0 ? '+' : ''}${analytics.bookings_change_percent.toFixed(1)}% vs last month`
        : 'This month',
      icon: Ticket,
      color: 'text-green-500',
      bgColor: 'bg-green-500/10',
      trend: analytics?.bookings_change_percent,
    },
    {
      title: 'Total Revenue',
      value: formatCurrency(analytics?.total_revenue || systemStats?.total_revenue || 0),
      subValue: analytics?.revenue_change_percent
        ? `${analytics.revenue_change_percent >= 0 ? '+' : ''}${analytics.revenue_change_percent.toFixed(1)}% vs last month`
        : 'This month',
      icon: DollarSign,
      color: 'text-amber-500',
      bgColor: 'bg-amber-500/10',
      trend: analytics?.revenue_change_percent,
    },
  ]

  return (
    <div data-testid="admin-stats-cards" className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
      {stats.map((stat) => (
        <Card key={stat.title} data-testid={`admin-stat-card-${stat.title.toLowerCase().replace(/\s+/g, '-')}`}>
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
            <p className={`text-xs flex items-center gap-1 ${
              stat.trend !== undefined
                ? stat.trend >= 0 ? 'text-green-500' : 'text-red-500'
                : 'text-muted-foreground'
            }`}>
              {stat.trend !== undefined && (
                stat.trend >= 0 ? (
                  <TrendingUp className="h-3 w-3" />
                ) : (
                  <TrendingDown className="h-3 w-3" />
                )
              )}
              {stat.subValue}
            </p>
          </CardContent>
        </Card>
      ))}
    </div>
  )
}

// Quick Links
function QuickLinks() {
  const links = [
    {
      title: 'Manage Users',
      description: 'View and manage user accounts',
      href: '/admin/users',
      icon: Users,
      color: 'text-blue-500',
    },
    {
      title: 'All Events',
      description: 'View all events across tenants',
      href: '/admin/events',
      icon: Calendar,
      color: 'text-purple-500',
    },
    {
      title: 'Tenants',
      description: 'Manage organization tenants',
      href: '/admin/tenants',
      icon: Building2,
      color: 'text-green-500',
    },
    {
      title: 'Analytics',
      description: 'System-wide analytics',
      href: '/admin/analytics',
      icon: Activity,
      color: 'text-amber-500',
    },
  ]

  return (
    <div data-testid="admin-quick-links" className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
      {links.map((link) => (
        <Link key={link.href} href={link.href}>
          <Card data-testid={`admin-quick-link-${link.title.toLowerCase().replace(/\s+/g, '-')}`} className="hover:border-primary/50 transition-colors cursor-pointer h-full">
            <CardHeader>
              <div className="flex items-center gap-3">
                <link.icon className={`h-5 w-5 ${link.color}`} />
                <CardTitle className="text-base">{link.title}</CardTitle>
              </div>
              <CardDescription>{link.description}</CardDescription>
            </CardHeader>
          </Card>
        </Link>
      ))}
    </div>
  )
}

// Recent Bookings
function RecentBookingsSection() {
  const { data, isLoading, error } = useRecentBookings(5)

  const statusColors: Record<string, string> = {
    confirmed: 'bg-green-500',
    completed: 'bg-blue-500',
    pending: 'bg-yellow-500',
    reserved: 'bg-orange-500',
    cancelled: 'bg-red-500',
    expired: 'bg-gray-500',
  }

  return (
    <Card data-testid="admin-recent-bookings">
      <CardHeader>
        <CardTitle>Recent Bookings</CardTitle>
        <CardDescription>Latest bookings across all events</CardDescription>
      </CardHeader>
      <CardContent>
        {isLoading ? (
          <div className="space-y-3">
            {[...Array(5)].map((_, i) => (
              <div key={i} className="flex items-center gap-3">
                <Skeleton className="h-2 w-2 rounded-full" />
                <Skeleton className="h-4 flex-1" />
                <Skeleton className="h-4 w-20" />
              </div>
            ))}
          </div>
        ) : error || !data || data.length === 0 ? (
          <div className="text-center text-muted-foreground py-4">
            No recent bookings
          </div>
        ) : (
          <div className="space-y-3">
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

// System Health (placeholder)
function SystemHealth() {
  const services = [
    { name: 'API Gateway', status: 'healthy', latency: '12ms' },
    { name: 'Auth Service', status: 'healthy', latency: '8ms' },
    { name: 'Booking Service', status: 'healthy', latency: '15ms' },
    { name: 'Payment Service', status: 'healthy', latency: '20ms' },
    { name: 'Notification Service', status: 'healthy', latency: '10ms' },
  ]

  return (
    <Card data-testid="admin-system-health">
      <CardHeader>
        <CardTitle>System Health</CardTitle>
        <CardDescription>Service status overview</CardDescription>
      </CardHeader>
      <CardContent>
        <div className="space-y-3">
          {services.map((service) => (
            <div
              key={service.name}
              className="flex items-center justify-between py-2 border-b last:border-0"
            >
              <div className="flex items-center gap-3">
                <div className={`h-2 w-2 rounded-full ${
                  service.status === 'healthy' ? 'bg-green-500' : 'bg-red-500'
                }`} />
                <span className="text-sm">{service.name}</span>
              </div>
              <span className="text-xs text-muted-foreground">
                {service.latency}
              </span>
            </div>
          ))}
        </div>
      </CardContent>
    </Card>
  )
}

// Main Admin Dashboard
export default function AdminDashboardPage() {
  return (
    <div data-testid="admin-dashboard-page" className="space-y-8">
      <div>
        <h1 data-testid="admin-dashboard-title" className="text-3xl font-bold">Admin Dashboard</h1>
        <p className="text-muted-foreground">
          System overview and management
        </p>
      </div>

      {/* System Stats */}
      <SystemStatsCards />

      {/* Quick Links */}
      <div>
        <h2 className="text-lg font-semibold mb-4">Quick Access</h2>
        <QuickLinks />
      </div>

      {/* Recent Activity & Health */}
      <div data-testid="admin-recent-activity" className="grid gap-4 lg:grid-cols-2">
        <RecentBookingsSection />
        <SystemHealth />
      </div>
    </div>
  )
}
