import axios, { AxiosInstance, AxiosError, InternalAxiosRequestConfig } from "axios"
import type { ApiError } from "./types"

const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080/api/v1"

class ApiClient {
  private axiosInstance: AxiosInstance
  private isRefreshing = false
  private failedQueue: Array<{
    resolve: (token: string) => void
    reject: (error: Error) => void
  }> = []

  constructor(baseUrl: string) {
    // Create Axios instance with base configuration
    this.axiosInstance = axios.create({
      baseURL: baseUrl,
      headers: {
        "Content-Type": "application/json",
      },
      withCredentials: true,
    })

    this.setupInterceptors()
  }

  private setupInterceptors() {
    // Request interceptor: Inject JWT token
    this.axiosInstance.interceptors.request.use(
      (config: InternalAxiosRequestConfig) => {
        const token = this.getAccessToken()
        if (token && config.headers) {
          config.headers.Authorization = `Bearer ${token}`
        }
        return config
      },
      (error) => {
        return Promise.reject(error)
      }
    )

    // Response interceptor: Handle errors and token refresh
    this.axiosInstance.interceptors.response.use(
      (response) => {
        // Backend wraps response in { success: boolean, data: T }
        const data = response.data

        if (data && typeof data === "object" && "success" in data) {
          // Paginated response with meta
          if ("meta" in data) {
            const result = { data: data.data, meta: data.meta }
            // Assign back to response.data and return response object
            response.data = result
            return response
          }
          // Regular response with data wrapper
          if ("data" in data) {
            // Assign back to response.data and return response object
            response.data = data.data
            return response
          }
        }
        return response
      },
      async (error: AxiosError<ApiError>) => {
        const originalRequest = error.config as InternalAxiosRequestConfig & { _retry?: boolean }

        // Handle 401 Unauthorized with token refresh
        if (error.response?.status === 401 && !originalRequest._retry) {
          if (this.isRefreshing) {
            // Queue the request while refresh is in progress
            return new Promise((resolve, reject) => {
              this.failedQueue.push({ resolve, reject })
            })
              .then((token) => {
                if (originalRequest.headers) {
                  originalRequest.headers.Authorization = `Bearer ${token}`
                }
                return this.axiosInstance(originalRequest)
              })
              .catch((err) => {
                return Promise.reject(err)
              })
          }

          originalRequest._retry = true
          this.isRefreshing = true

          try {
            const refreshToken = this.getRefreshToken()
            if (!refreshToken) {
              throw new Error("No refresh token available")
            }

            // Attempt to refresh the token
            const response = await axios.post(
              `${API_BASE_URL}/auth/refresh`,
              { refresh_token: refreshToken },
              { withCredentials: true }
            )

            const { access_token, refresh_token: newRefreshToken, user } = response.data.data || response.data

            // Update tokens
            this.setAccessToken(access_token)
            if (typeof window !== "undefined") {
              localStorage.setItem("refresh_token", newRefreshToken)
              localStorage.setItem("user", JSON.stringify(user))
            }

            // Process queued requests
            this.failedQueue.forEach((promise) => promise.resolve(access_token))
            this.failedQueue = []

            // Retry original request
            if (originalRequest.headers) {
              originalRequest.headers.Authorization = `Bearer ${access_token}`
            }
            return this.axiosInstance(originalRequest)
          } catch (refreshError) {
            // Refresh failed - clear tokens and redirect
            this.failedQueue.forEach((promise) => promise.reject(refreshError as Error))
            this.failedQueue = []
            this.clearTokens()

            if (typeof window !== "undefined") {
              window.dispatchEvent(new CustomEvent("auth:unauthorized"))
            }

            return Promise.reject(refreshError)
          } finally {
            this.isRefreshing = false
          }
        }

        // Handle other errors
        const errorData = error.response?.data || {
          error: error.message,
          message: error.message,
          code: undefined,
        }

        // Extract error message - handle nested error object from backend
        let errorMessage = error.message
        let errorCode: string | undefined

        if (errorData.error && typeof errorData.error === "object") {
          // Backend returns { success: false, error: { code, message } }
          errorMessage = errorData.error.message || error.message
          errorCode = errorData.error.code
        } else if (errorData.message) {
          errorMessage = errorData.message
          errorCode = errorData.code
        } else if (typeof errorData.error === "string") {
          errorMessage = errorData.error
          errorCode = errorData.code
        }

        throw new ApiRequestError(
          errorMessage,
          error.response?.status || 500,
          errorCode
        )
      }
    )
  }

  setAccessToken(token: string | null) {
    if (typeof window !== "undefined") {
      if (token) {
        localStorage.setItem("access_token", token)
      } else {
        localStorage.removeItem("access_token")
      }
    }
  }

  getAccessToken(): string | null {
    if (typeof window !== "undefined") {
      return localStorage.getItem("access_token")
    }
    return null
  }

  getRefreshToken(): string | null {
    if (typeof window !== "undefined") {
      return localStorage.getItem("refresh_token")
    }
    return null
  }

  clearTokens() {
    if (typeof window !== "undefined") {
      localStorage.removeItem("access_token")
      localStorage.removeItem("refresh_token")
      localStorage.removeItem("user")
    }
  }

  // Convenience methods
  async get<T>(endpoint: string, config = {}): Promise<T> {
    const response = await this.axiosInstance.get<T>(endpoint, config)
    return response.data
  }

  async post<T>(endpoint: string, data?: unknown, config = {}): Promise<T> {
    const response = await this.axiosInstance.post<T>(endpoint, data, config)
    return response.data
  }

  async put<T>(endpoint: string, data?: unknown, config = {}): Promise<T> {
    const response = await this.axiosInstance.put<T>(endpoint, data, config)
    return response.data
  }

  async patch<T>(endpoint: string, data?: unknown, config = {}): Promise<T> {
    const response = await this.axiosInstance.patch<T>(endpoint, data, config)
    return response.data
  }

  async delete<T>(endpoint: string, config = {}): Promise<T> {
    const response = await this.axiosInstance.delete<T>(endpoint, config)
    return response.data
  }

  // Get the axios instance for advanced usage
  getAxiosInstance(): AxiosInstance {
    return this.axiosInstance
  }
}

export class ApiRequestError extends Error {
  status: number
  code?: string

  constructor(message: string, status: number, code?: string) {
    super(message)
    this.name = "ApiRequestError"
    this.status = status
    this.code = code
  }
}

export const apiClient = new ApiClient(API_BASE_URL)
