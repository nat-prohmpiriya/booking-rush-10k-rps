"use client"

import { Header } from "@/components/header"
import { Rocket, Clock } from "lucide-react"

export default function AboutPage() {
  return (
    <main className="min-h-screen bg-background">
      <Header />

      {/* Hero Section */}
      <section className="relative pt-24 pb-12 lg:pt-32 lg:pb-16 overflow-hidden">
        {/* Background */}
        <div className="absolute inset-0 bg-[radial-gradient(ellipse_at_top,var(--tw-gradient-stops))] from-primary/20 via-background to-background" />
        <div className="absolute inset-0 bg-[url('/images/grid-pattern.svg')] opacity-5" />

        <div className="container mx-auto px-4 lg:px-8 relative z-10">
          <div className="max-w-3xl mx-auto text-center space-y-6">
            <div className="inline-block glass px-4 py-2 rounded-full">
              <span className="text-primary text-sm font-medium flex items-center gap-2">
                <Rocket className="h-4 w-4" />
                Company
              </span>
            </div>
            <h1 className="text-4xl lg:text-5xl font-bold text-balance">
              About{" "}
              <span className="bg-linear-to-r from-primary to-amber-400 bg-clip-text text-transparent">
                Us
              </span>
            </h1>
          </div>
        </div>
      </section>

      {/* Coming Soon Content */}
      <section className="container mx-auto px-4 lg:px-8 pb-16 lg:pb-24">
        <div className="max-w-2xl mx-auto">
          <div className="glass rounded-xl border border-primary/30 p-12 text-center">
            <div className="inline-block p-6 rounded-full bg-primary/20 mb-6">
              <Clock className="h-12 w-12 text-primary" />
            </div>
            <h2 className="text-3xl font-bold text-foreground mb-4">Coming Soon</h2>
            <p className="text-muted-foreground text-lg leading-relaxed">
              We&apos;re working on something exciting! Our story, mission, and the team behind BookingRush will be shared here soon.
            </p>
          </div>
        </div>
      </section>
    </main>
  )
}
