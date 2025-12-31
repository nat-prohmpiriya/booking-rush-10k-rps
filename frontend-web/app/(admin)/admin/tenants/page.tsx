'use client'

import { useState } from 'react'
import { useTenants } from '@/hooks/use-admin'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
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
import { Search, ChevronLeft, ChevronRight, Building2, Calendar, Ticket } from 'lucide-react'

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

function formatNumber(num: number): string {
  return new Intl.NumberFormat('th-TH').format(num)
}

const statusColors: Record<string, string> = {
  active: 'bg-green-500',
  inactive: 'bg-gray-500',
  suspended: 'bg-red-500',
}

export default function TenantsPage() {
  const [search, setSearch] = useState('')
  const [status, setStatus] = useState<string>('')
  const [page, setPage] = useState(1)
  const limit = 10

  const { data, isLoading, error, refetch } = useTenants({
    search: search || undefined,
    status: status || undefined,
    page,
    limit,
  })

  const handleSearch = (e: React.FormEvent) => {
    e.preventDefault()
    setPage(1)
    refetch()
  }

  return (
    <div data-testid="admin-tenants-page" className="space-y-6">
      <div>
        <h1 data-testid="admin-tenants-title" className="text-3xl font-bold">Tenants</h1>
        <p className="text-muted-foreground">
          Manage organization tenants
        </p>
      </div>

      {/* Filters */}
      <Card data-testid="admin-tenants-filters">
        <CardContent className="pt-6">
          <form onSubmit={handleSearch} className="flex gap-4">
            <div className="relative flex-1">
              <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
              <Input
                data-testid="admin-tenants-search"
                placeholder="Search by name..."
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                className="pl-10"
              />
            </div>
            <Select value={status} onValueChange={(v) => { setStatus(v); setPage(1); }}>
              <SelectTrigger data-testid="admin-tenants-filter" className="w-40">
                <SelectValue placeholder="All status" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="">All status</SelectItem>
                <SelectItem value="active">Active</SelectItem>
                <SelectItem value="inactive">Inactive</SelectItem>
                <SelectItem value="suspended">Suspended</SelectItem>
              </SelectContent>
            </Select>
            <Button data-testid="admin-tenants-search-button" type="submit">Search</Button>
          </form>
        </CardContent>
      </Card>

      {/* Tenants Table */}
      <Card data-testid="admin-tenants-table-card">
        <CardHeader>
          <CardTitle>All Tenants</CardTitle>
          <CardDescription>
            {data?.meta?.total || 0} tenants found
          </CardDescription>
        </CardHeader>
        <CardContent>
          {isLoading ? (
            <div className="space-y-4">
              {[...Array(5)].map((_, i) => (
                <div key={i} className="flex items-center gap-4">
                  <Skeleton className="h-10 w-10 rounded" />
                  <div className="flex-1">
                    <Skeleton className="h-4 w-40 mb-2" />
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
              No tenants found
            </div>
          ) : (
            <>
              <Table data-testid="admin-tenants-table">
                <TableHeader>
                  <TableRow>
                    <TableHead>Tenant</TableHead>
                    <TableHead>Owner</TableHead>
                    <TableHead>Events</TableHead>
                    <TableHead>Bookings</TableHead>
                    <TableHead>Revenue</TableHead>
                    <TableHead>Status</TableHead>
                    <TableHead>Created</TableHead>
                    <TableHead className="text-right">Actions</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {data.data.map((tenant) => (
                    <TableRow key={tenant.id}>
                      <TableCell>
                        <div className="flex items-center gap-3">
                          <div className="h-10 w-10 rounded bg-muted flex items-center justify-center">
                            <Building2 className="h-5 w-5 text-muted-foreground" />
                          </div>
                          <div>
                            <p className="font-medium">{tenant.name}</p>
                            <p className="text-xs text-muted-foreground">
                              {tenant.slug}
                            </p>
                          </div>
                        </div>
                      </TableCell>
                      <TableCell className="text-muted-foreground text-sm">
                        {tenant.owner_email}
                      </TableCell>
                      <TableCell>
                        <div className="flex items-center gap-1">
                          <Calendar className="h-3 w-3 text-muted-foreground" />
                          <span>{formatNumber(tenant.total_events)}</span>
                        </div>
                      </TableCell>
                      <TableCell>
                        <div className="flex items-center gap-1">
                          <Ticket className="h-3 w-3 text-muted-foreground" />
                          <span>{formatNumber(tenant.total_bookings)}</span>
                        </div>
                      </TableCell>
                      <TableCell className="font-medium">
                        {formatCurrency(tenant.total_revenue)}
                      </TableCell>
                      <TableCell>
                        <Badge className={`${statusColors[tenant.status] || 'bg-gray-500'} text-white`}>
                          {tenant.status}
                        </Badge>
                      </TableCell>
                      <TableCell className="text-muted-foreground text-sm">
                        {formatDate(tenant.created_at)}
                      </TableCell>
                      <TableCell className="text-right">
                        <Button variant="ghost" size="sm">
                          View
                        </Button>
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>

              {/* Pagination */}
              {data.meta && data.meta.total_pages > 1 && (
                <div data-testid="admin-tenants-pagination" className="flex items-center justify-between mt-4">
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
