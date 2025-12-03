import { ArrowLeft, HelpCircle } from "lucide-react"
import { Button } from "@/components/ui/button"
import Link from "next/link"

export function BookingHeader() {
  return (
    <header className="sticky top-0 z-50 border-b border-border bg-background/95 backdrop-blur supports-[backdrop-filter]:bg-background/60">
      <div className="container mx-auto flex h-14 items-center justify-between px-4">
        <div className="flex items-center gap-4">
          <Button variant="ghost" size="icon" asChild>
            <Link href="/">
              <ArrowLeft className="h-5 w-5" />
              <span className="sr-only">Go back</span>
            </Link>
          </Button>
          <div className="hidden sm:block">
            <span className="text-sm font-medium text-foreground">NEON NIGHTS</span>
            <span className="ml-2 text-xs text-muted-foreground">WORLD TOUR 2025</span>
          </div>
        </div>

        <nav className="flex items-center gap-2">
          <div className="mr-4 hidden items-center gap-6 text-sm md:flex">
            <span className="text-muted-foreground">1. Event</span>
            <span className="text-muted-foreground">2. Tickets</span>
            <span className="font-medium text-foreground">3. Seats</span>
            <span className="text-muted-foreground">4. Payment</span>
          </div>
          <Button variant="ghost" size="sm" className="gap-2">
            <HelpCircle className="h-4 w-4" />
            <span className="hidden sm:inline">Help</span>
          </Button>
        </nav>
      </div>
    </header>
  )
}
