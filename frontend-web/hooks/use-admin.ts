'use client'

import { useState, useEffect, useCallback } from 'react'
import { adminApi, SystemStatsResponse, UserListFilter, TenantListFilter, TenantResponse } from '@/lib/api/admin'
import type { UserResponse, EventResponse, PaginatedResponse } from '@/lib/api/types'

/**
 * Hook for fetching system statistics
 */
export function useSystemStats() {
  const [data, setData] = useState<SystemStatsResponse | null>(null)
  const [isLoading, setIsLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const fetchData = useCallback(async () => {
    try {
      setIsLoading(true)
      setError(null)
      const result = await adminApi.getSystemStats()
      setData(result)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to fetch system stats')
      // Set mock data for development
      setData({
        total_users: 0,
        total_events: 0,
        total_bookings: 0,
        total_revenue: 0,
        users_this_month: 0,
        events_this_month: 0,
        bookings_this_month: 0,
        revenue_this_month: 0,
      })
    } finally {
      setIsLoading(false)
    }
  }, [])

  useEffect(() => {
    fetchData()
  }, [fetchData])

  return { data, isLoading, error, refetch: fetchData }
}

/**
 * Hook for fetching users list
 */
export function useUsers(filter?: UserListFilter) {
  const [data, setData] = useState<PaginatedResponse<UserResponse> | null>(null)
  const [isLoading, setIsLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const fetchData = useCallback(async () => {
    try {
      setIsLoading(true)
      setError(null)
      const result = await adminApi.listUsers(filter)
      setData(result)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to fetch users')
      // Set empty data for development
      setData({
        data: [],
        meta: { page: 1, per_page: 10, total: 0, total_pages: 0 }
      })
    } finally {
      setIsLoading(false)
    }
  }, [filter?.search, filter?.role, filter?.status, filter?.page, filter?.limit])

  useEffect(() => {
    fetchData()
  }, [fetchData])

  return { data, isLoading, error, refetch: fetchData }
}

/**
 * Hook for fetching all events (admin view)
 */
export function useAllEvents(filter?: { page?: number; limit?: number; status?: string }) {
  const [data, setData] = useState<PaginatedResponse<EventResponse> | null>(null)
  const [isLoading, setIsLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const fetchData = useCallback(async () => {
    try {
      setIsLoading(true)
      setError(null)
      const result = await adminApi.listAllEvents(filter)
      setData(result)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to fetch events')
      setData({
        data: [],
        meta: { page: 1, per_page: 10, total: 0, total_pages: 0 }
      })
    } finally {
      setIsLoading(false)
    }
  }, [filter?.page, filter?.limit, filter?.status])

  useEffect(() => {
    fetchData()
  }, [fetchData])

  return { data, isLoading, error, refetch: fetchData }
}

/**
 * Hook for fetching tenants
 */
export function useTenants(filter?: TenantListFilter) {
  const [data, setData] = useState<PaginatedResponse<TenantResponse> | null>(null)
  const [isLoading, setIsLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const fetchData = useCallback(async () => {
    try {
      setIsLoading(true)
      setError(null)
      const result = await adminApi.listTenants(filter)
      setData(result)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to fetch tenants')
      setData({
        data: [],
        meta: { page: 1, per_page: 10, total: 0, total_pages: 0 }
      })
    } finally {
      setIsLoading(false)
    }
  }, [filter?.search, filter?.status, filter?.page, filter?.limit])

  useEffect(() => {
    fetchData()
  }, [fetchData])

  return { data, isLoading, error, refetch: fetchData }
}
