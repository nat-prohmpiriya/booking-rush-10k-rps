"use client"

import { useEffect, useState, useCallback, useRef, Suspense } from "react"
import { Card } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { useRouter, useSearchParams } from "next/navigation"
import { queueApi } from "@/lib/api/booking"
import type { QueuePositionResponse, JoinQueueResponse } from "@/lib/api/types"
import { ApiRequestError } from "@/lib/api/client"

type QueueState = "joining" | "waiting" | "ready" | "error"

function QueueWaitingRoomContent() {
  const router = useRouter()
  const searchParams = useSearchParams()

  const eventId = searchParams.get("event_id")
  const showId = searchParams.get("show_id")
  const ticketsParam = searchParams.get("tickets")
  const totalParam = searchParams.get("total")

  const [queueState, setQueueState] = useState<QueueState>("joining")
  const [position, setPosition] = useState<number>(0)
  const [totalInQueue, setTotalInQueue] = useState<number>(0)
  const [estimatedWait, setEstimatedWait] = useState<number>(0)
  const [queueToken, setQueueToken] = useState<string>("")
  const [queuePass, setQueuePass] = useState<string>("")
  const [queuePassExpiresAt, setQueuePassExpiresAt] = useState<string>("")
  const [error, setError] = useState<string>("")
  const [dots, setDots] = useState("")
  const hasJoinedRef = useRef(false)

  // Parse ticket selection
  const selectedTickets = ticketsParam ? JSON.parse(ticketsParam) : {}
  const totalPrice = totalParam ? parseInt(totalParam, 10) : 0

  // Join queue on mount
  useEffect(() => {
    if (!eventId) {
      setError("No event selected")
      setQueueState("error")
      return
    }

    // Prevent double call in Strict Mode (React 19)
    if (hasJoinedRef.current) return
    hasJoinedRef.current = true

    const joinQueue = async () => {
      try {
        const response: JoinQueueResponse = await queueApi.joinQueue({ event_id: eventId })
        setQueueToken(response.token)
        setPosition(response.position)
        setEstimatedWait(response.estimated_wait_seconds)
        setQueueState("waiting")

        // Store queue data in sessionStorage
        sessionStorage.setItem("queue_token", response.token)
        sessionStorage.setItem("queue_event_id", eventId)
        if (showId) sessionStorage.setItem("queue_show_id", showId)
        sessionStorage.setItem("queue_tickets", ticketsParam || "{}")
        sessionStorage.setItem("queue_total", totalParam || "0")
      } catch (err) {
        if (err instanceof ApiRequestError) {
          if (err.code === "ALREADY_IN_QUEUE") {
            // Already in queue, just start polling
            setQueueState("waiting")
            return
          }
          setError(err.message)
        } else {
          setError("Failed to join queue")
        }
        setQueueState("error")
      }
    }

    joinQueue()
  }, [eventId, showId, ticketsParam, totalParam])

  // Poll for queue position
  const pollPosition = useCallback(async () => {
    if (!eventId || queueState !== "waiting") return

    try {
      const response: QueuePositionResponse = await queueApi.getPosition(eventId)
      setPosition(response.position)
      setTotalInQueue(response.total_in_queue)
      setEstimatedWait(response.estimated_wait_seconds)

      if (response.is_ready && response.queue_pass) {
        setQueuePass(response.queue_pass)
        setQueuePassExpiresAt(response.queue_pass_expires_at || "")
        setQueueState("ready")

        // Store queue pass for checkout
        sessionStorage.setItem("queue_pass", response.queue_pass)
        sessionStorage.setItem("queue_pass_expires_at", response.queue_pass_expires_at || "")
      }
    } catch (err) {
      console.error("Failed to poll queue position:", err)
    }
  }, [eventId, queueState])

  // Poll every 3 seconds
  useEffect(() => {
    if (queueState !== "waiting") return

    const interval = setInterval(pollPosition, 3000)
    // Initial poll
    pollPosition()

    return () => clearInterval(interval)
  }, [queueState, pollPosition])

  // Auto-redirect when ready
  useEffect(() => {
    if (queueState === "ready" && queuePass) {
      // Redirect to checkout after short delay
      const timeout = setTimeout(() => {
        router.push("/checkout")
      }, 2000)
      return () => clearTimeout(timeout)
    }
  }, [queueState, queuePass, router])

  // Animate loading dots
  useEffect(() => {
    const interval = setInterval(() => {
      setDots((prev) => (prev.length >= 3 ? "" : prev + "."))
    }, 500)
    return () => clearInterval(interval)
  }, [])

  // Format estimated wait time
  const formatWaitTime = (seconds: number) => {
    if (seconds < 60) return `${seconds} seconds`
    const minutes = Math.ceil(seconds / 60)
    return `${minutes} ${minutes === 1 ? "minute" : "minutes"}`
  }

  // Leave queue handler
  const handleLeaveQueue = async () => {
    if (!eventId || !queueToken) {
      router.push("/")
      return
    }

    try {
      await queueApi.leaveQueue({ event_id: eventId, token: queueToken })
    } catch (err) {
      console.error("Failed to leave queue:", err)
    }

    // Clear session storage
    sessionStorage.removeItem("queue_token")
    sessionStorage.removeItem("queue_event_id")
    sessionStorage.removeItem("queue_show_id")
    sessionStorage.removeItem("queue_tickets")
    sessionStorage.removeItem("queue_total")

    router.push("/")
  }

  // Error state
  if (queueState === "error") {
    return (
      <div className="min-h-screen bg-[#0a0a0a] text-white flex items-center justify-center p-4">
        <Card className="bg-[#1a1a1a] border-red-800 p-8 max-w-md text-center">
          <div className="w-16 h-16 rounded-full border-2 border-red-500 flex items-center justify-center mx-auto mb-4">
            <svg className="w-8 h-8 text-red-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
            </svg>
          </div>
          <h1 className="text-2xl font-bold mb-2">Unable to Join Queue</h1>
          <p className="text-gray-400 mb-6">{error}</p>
          <Button onClick={() => router.push("/")} className="bg-[#d4af37] hover:bg-[#d4af37]/90 text-black">
            Back to Events
          </Button>
        </Card>
      </div>
    )
  }

  // Ready state
  if (queueState === "ready") {
    return (
      <div className="min-h-screen bg-[#0a0a0a] text-white flex items-center justify-center p-4">
        <div className="w-full max-w-2xl space-y-8 text-center">
          <div className="space-y-4">
            <div className="w-20 h-20 rounded-full bg-green-500/20 border-2 border-green-500 flex items-center justify-center mx-auto">
              <svg className="w-10 h-10 text-green-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
              </svg>
            </div>
            <h1 className="text-4xl md:text-5xl font-bold">It's Your Turn!</h1>
            <p className="text-gray-400 text-lg">Redirecting to checkout{dots}</p>
          </div>

          <div className="py-4">
            <div className="w-full h-2 bg-gray-800 rounded-full overflow-hidden">
              <div className="h-full bg-green-500 animate-pulse" style={{ width: "100%" }} />
            </div>
          </div>

          <Card className="bg-[#1a1a1a] border-green-800/50 p-6">
            <p className="text-sm text-gray-400 mb-2">Your Queue Pass expires in</p>
            <p className="text-2xl font-bold text-green-400">5:00</p>
            <p className="text-xs text-gray-500 mt-2">Complete your booking before it expires</p>
          </Card>
        </div>
      </div>
    )
  }

  // Joining/Waiting state
  return (
    <div className="min-h-screen bg-[#0a0a0a] text-white flex items-center justify-center p-4">
      <div className="w-full max-w-2xl space-y-8">
        {/* Main Content */}
        <div className="text-center space-y-8">
          {/* Header */}
          <div className="space-y-4">
            <div className="inline-block">
              <div className="w-16 h-16 rounded-full border-2 border-[#d4af37] flex items-center justify-center mx-auto mb-4">
                <svg className="w-8 h-8 text-[#d4af37]" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    strokeWidth={2}
                    d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z"
                  />
                </svg>
              </div>
            </div>

            <h1 className="text-4xl md:text-5xl font-bold text-balance">
              {queueState === "joining" ? "Joining Queue" : "You're in the Queue"}
            </h1>

            <p className="text-gray-400 text-lg">Thank you for your patience</p>
          </div>

          {/* Position Display */}
          {queueState === "waiting" && (
            <>
              <div className="space-y-2">
                <p className="text-sm uppercase tracking-wider text-[#d4af37] font-medium">Your Position</p>
                <div className="relative">
                  <div className="text-7xl md:text-8xl font-bold text-[#d4af37] animate-pulse">
                    #{position.toLocaleString()}
                  </div>
                  {/* Glow effect */}
                  <div className="absolute inset-0 blur-3xl opacity-30 bg-[#d4af37] -z-10" />
                </div>
                {totalInQueue > 0 && (
                  <p className="text-sm text-gray-500">of {totalInQueue.toLocaleString()} in queue</p>
                )}
              </div>

              {/* Estimated Wait Time */}
              <div className="space-y-2">
                <p className="text-sm uppercase tracking-wider text-gray-400 font-medium">Estimated Wait Time</p>
                <p className="text-3xl font-semibold text-white">~{formatWaitTime(estimatedWait)}</p>
              </div>
            </>
          )}

          {/* Loading state for joining */}
          {queueState === "joining" && (
            <div className="py-8">
              <div className="w-16 h-16 border-4 border-[#d4af37]/30 border-t-[#d4af37] rounded-full animate-spin mx-auto" />
            </div>
          )}

          {/* Animated Progress Indicator */}
          <div className="py-8">
            <div className="w-full h-1 bg-gray-800 rounded-full overflow-hidden">
              <div
                className="h-full bg-gradient-to-r from-[#d4af37] to-[#f4d03f]"
                style={{
                  width: "100%",
                  animation: "shimmer 2s ease-in-out infinite",
                }}
              />
            </div>
          </div>

          {/* Keep Page Open Notice */}
          <Card className="bg-[#1a1a1a] border-gray-800 p-6">
            <div className="flex items-start gap-4">
              <div className="shrink-0">
                <svg className="w-6 h-6 text-[#d4af37]" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    strokeWidth={2}
                    d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
                  />
                </svg>
              </div>
              <div className="text-left space-y-1">
                <p className="font-semibold text-white">Please keep this page open</p>
                <p className="text-sm text-gray-400">
                  Closing this page will lose your spot in the queue. You'll be automatically redirected when it's your
                  turn.
                </p>
              </div>
            </div>
          </Card>
        </div>

        {/* Order Summary Card */}
        {Object.keys(selectedTickets).length > 0 && (
          <Card className="bg-gradient-to-br from-[#1a1a1a] to-[#0f0f0f] border-[#d4af37]/20 p-6 md:p-8">
            <div className="space-y-4">
              <div className="flex items-center gap-2">
                <div className="w-2 h-2 rounded-full bg-[#d4af37] animate-pulse" />
                <p className="text-xs uppercase tracking-wider text-[#d4af37] font-medium">Your Order</p>
              </div>

              <div className="space-y-2">
                {Object.entries(selectedTickets).map(([zoneId, quantity]) => (
                  <div key={zoneId} className="flex justify-between text-sm">
                    <span className="text-gray-400">Zone ticket x{quantity as number}</span>
                  </div>
                ))}
              </div>

              <div className="pt-4 border-t border-gray-800">
                <div className="flex justify-between items-center">
                  <span className="text-gray-400">Total</span>
                  <span className="text-2xl font-bold text-[#d4af37]">฿{totalPrice.toLocaleString()}</span>
                </div>
              </div>
            </div>
          </Card>
        )}

        {/* Auto-refresh Indicator */}
        <div className="flex items-center justify-center gap-2 text-sm text-gray-500">
          <div className="w-2 h-2 rounded-full bg-green-500 animate-pulse" />
          <span>Auto-updating{dots}</span>
        </div>

        {/* Leave Queue Button */}
        <div className="text-center">
          <Button
            variant="ghost"
            onClick={handleLeaveQueue}
            className="text-gray-500 hover:text-gray-300"
          >
            Leave Queue
          </Button>
        </div>
      </div>

      {/* Custom animation styles */}
      <style jsx>{`
        @keyframes shimmer {
          0%, 100% {
            transform: translateX(-100%);
          }
          50% {
            transform: translateX(100%);
          }
        }
      `}</style>
    </div>
  )
}

export default function QueueWaitingRoom() {
  return (
    <Suspense fallback={
      <div className="min-h-screen bg-[#0a0a0a] text-white flex items-center justify-center">
        <div className="w-16 h-16 border-4 border-[#d4af37]/30 border-t-[#d4af37] rounded-full animate-spin" />
      </div>
    }>
      <QueueWaitingRoomContent />
    </Suspense>
  )
}
