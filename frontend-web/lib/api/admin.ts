import { apiClient } from './client'
import type { ApiResponse, PaginatedResponse, UserResponse, EventResponse } from './types'

// Admin-specific types
export interface SystemStatsResponse {
  total_users: number
  total_events: number
  total_bookings: number
  total_revenue: number
  users_this_month: number
  events_this_month: number
  bookings_this_month: number
  revenue_this_month: number
}

export interface UserListFilter {
  search?: string
  role?: string
  status?: string
  page?: number
  limit?: number
}

export interface UpdateUserRoleRequest {
  role: string
}

export interface TenantResponse {
  id: string
  name: string
  slug: string
  status: string
  owner_id: string
  owner_email: string
  total_events: number
  total_bookings: number
  total_revenue: number
  created_at: string
}

export interface TenantListFilter {
  search?: string
  status?: string
  page?: number
  limit?: number
}

/**
 * Admin API client
 * For super_admin and admin roles only
 */
export const adminApi = {
  /**
   * Get system-wide statistics
   */
  async getSystemStats(): Promise<SystemStatsResponse> {
    const response = await apiClient.get<ApiResponse<SystemStatsResponse>>('/admin/stats')
    return response.data!
  },

  /**
   * List all users (admin only)
   */
  async listUsers(filter?: UserListFilter): Promise<PaginatedResponse<UserResponse>> {
    const params = new URLSearchParams()
    if (filter?.search) params.append('search', filter.search)
    if (filter?.role) params.append('role', filter.role)
    if (filter?.status) params.append('status', filter.status)
    if (filter?.page) params.append('page', String(filter.page))
    if (filter?.limit) params.append('limit', String(filter.limit))

    const response = await apiClient.get<PaginatedResponse<UserResponse>>(
      `/admin/users?${params.toString()}`
    )
    return response
  },

  /**
   * Get user by ID
   */
  async getUser(userId: string): Promise<UserResponse> {
    const response = await apiClient.get<ApiResponse<UserResponse>>(`/admin/users/${userId}`)
    return response.data!
  },

  /**
   * Update user role
   */
  async updateUserRole(userId: string, data: UpdateUserRoleRequest): Promise<UserResponse> {
    const response = await apiClient.put<ApiResponse<UserResponse>>(
      `/admin/users/${userId}/role`,
      data
    )
    return response.data!
  },

  /**
   * List all events (admin view)
   */
  async listAllEvents(filter?: { page?: number; limit?: number; status?: string }): Promise<PaginatedResponse<EventResponse>> {
    const params = new URLSearchParams()
    if (filter?.page) params.append('page', String(filter.page))
    if (filter?.limit) params.append('limit', String(filter.limit))
    if (filter?.status) params.append('status', filter.status)

    const response = await apiClient.get<PaginatedResponse<EventResponse>>(
      `/admin/events?${params.toString()}`
    )
    return response
  },

  /**
   * List all tenants
   */
  async listTenants(filter?: TenantListFilter): Promise<PaginatedResponse<TenantResponse>> {
    const params = new URLSearchParams()
    if (filter?.search) params.append('search', filter.search)
    if (filter?.status) params.append('status', filter.status)
    if (filter?.page) params.append('page', String(filter.page))
    if (filter?.limit) params.append('limit', String(filter.limit))

    const response = await apiClient.get<PaginatedResponse<TenantResponse>>(
      `/admin/tenants?${params.toString()}`
    )
    return response
  },

  /**
   * Get tenant by ID
   */
  async getTenant(tenantId: string): Promise<TenantResponse> {
    const response = await apiClient.get<ApiResponse<TenantResponse>>(`/admin/tenants/${tenantId}`)
    return response.data!
  },
}
