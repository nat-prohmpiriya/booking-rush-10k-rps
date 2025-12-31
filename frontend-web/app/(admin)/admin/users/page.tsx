'use client'

import { useState } from 'react'
import { useUsers } from '@/hooks/use-admin'
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
import { Search, ChevronLeft, ChevronRight, User } from 'lucide-react'

function formatDate(dateString: string): string {
  return new Date(dateString).toLocaleDateString('th-TH', {
    day: 'numeric',
    month: 'short',
    year: 'numeric',
  })
}

const roleColors: Record<string, string> = {
  super_admin: 'bg-red-500',
  admin: 'bg-orange-500',
  organizer: 'bg-blue-500',
  customer: 'bg-gray-500',
}

const roleLabels: Record<string, string> = {
  super_admin: 'Super Admin',
  admin: 'Admin',
  organizer: 'Organizer',
  customer: 'Customer',
}

export default function UsersPage() {
  const [search, setSearch] = useState('')
  const [role, setRole] = useState<string>('')
  const [page, setPage] = useState(1)
  const limit = 10

  const { data, isLoading, error, refetch } = useUsers({
    search: search || undefined,
    role: role || undefined,
    page,
    limit,
  })

  const handleSearch = (e: React.FormEvent) => {
    e.preventDefault()
    setPage(1)
    refetch()
  }

  return (
    <div data-testid="admin-users-page" className="space-y-6">
      <div>
        <h1 data-testid="admin-users-title" className="text-3xl font-bold">Users</h1>
        <p className="text-muted-foreground">
          Manage user accounts and roles
        </p>
      </div>

      {/* Filters */}
      <Card data-testid="admin-users-filters">
        <CardContent className="pt-6">
          <form onSubmit={handleSearch} className="flex gap-4">
            <div className="relative flex-1">
              <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
              <Input
                data-testid="admin-users-search"
                placeholder="Search by name or email..."
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                className="pl-10"
              />
            </div>
            <Select value={role} onValueChange={(v) => { setRole(v); setPage(1); }}>
              <SelectTrigger data-testid="admin-users-filter" className="w-40">
                <SelectValue placeholder="All roles" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="">All roles</SelectItem>
                <SelectItem value="super_admin">Super Admin</SelectItem>
                <SelectItem value="admin">Admin</SelectItem>
                <SelectItem value="organizer">Organizer</SelectItem>
                <SelectItem value="customer">Customer</SelectItem>
              </SelectContent>
            </Select>
            <Button type="submit">Search</Button>
          </form>
        </CardContent>
      </Card>

      {/* Users Table */}
      <Card data-testid="admin-users-table-card">
        <CardHeader>
          <CardTitle>All Users</CardTitle>
          <CardDescription>
            {data?.meta?.total || 0} users found
          </CardDescription>
        </CardHeader>
        <CardContent>
          {isLoading ? (
            <div className="space-y-4">
              {[...Array(5)].map((_, i) => (
                <div key={i} className="flex items-center gap-4">
                  <Skeleton className="h-10 w-10 rounded-full" />
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
              No users found
            </div>
          ) : (
            <>
              <Table data-testid="admin-users-table">
                <TableHeader>
                  <TableRow>
                    <TableHead>User</TableHead>
                    <TableHead>Email</TableHead>
                    <TableHead>Role</TableHead>
                    <TableHead>Joined</TableHead>
                    <TableHead className="text-right">Actions</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {data.data.map((user) => (
                    <TableRow key={user.id}>
                      <TableCell>
                        <div className="flex items-center gap-3">
                          <div className="h-10 w-10 rounded-full bg-muted flex items-center justify-center">
                            <User className="h-5 w-5 text-muted-foreground" />
                          </div>
                          <span className="font-medium">{user.name}</span>
                        </div>
                      </TableCell>
                      <TableCell className="text-muted-foreground">
                        {user.email}
                      </TableCell>
                      <TableCell>
                        <Badge className={`${roleColors[user.role] || 'bg-gray-500'} text-white`}>
                          {roleLabels[user.role] || user.role}
                        </Badge>
                      </TableCell>
                      <TableCell className="text-muted-foreground">
                        {formatDate(user.created_at)}
                      </TableCell>
                      <TableCell className="text-right">
                        <Button variant="ghost" size="sm">
                          Edit
                        </Button>
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>

              {/* Pagination */}
              {data.meta && data.meta.total_pages > 1 && (
                <div data-testid="admin-users-pagination" className="flex items-center justify-between mt-4">
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
