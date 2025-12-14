export function Logo({ className = "" }: { className?: string }) {
  return (
    <div className={`flex items-center gap-2 ${className}`}>
      <span className="text-4xl">🎫</span>
      <span className="text-2xl font-bold bg-linear-to-r from-primary via-primary/80 to-primary bg-clip-text text-transparent">
        BookingRush
      </span>
    </div>
  )
}
