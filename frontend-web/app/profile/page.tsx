"use client"

import { useState, useEffect } from "react"
import { useRouter } from "next/navigation"
import Link from "next/link"
import { Header } from "@/components/header"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Skeleton } from "@/components/ui/skeleton"
import { Badge } from "@/components/ui/badge"
import { Separator } from "@/components/ui/separator"
import { useAuth } from "@/contexts/auth-context"
import { paymentApi } from "@/lib/api/payment"
import {
  User,
  Mail,
  Calendar,
  Shield,
  Ticket,
  Settings,
  Bell,
  CreditCard,
  LogOut,
  ChevronRight,
  Edit3,
  Camera,
  Check,
  X,
  Lock,
  Globe,
  Phone,
  MapPin,
  Loader2,
} from "lucide-react"

interface ProfileFormData {
  name: string
  email: string
  phone: string
  location: string
}

function ProfileSkeleton() {
  return (
    <div className="space-y-8">
      {/* Avatar Skeleton */}
      <div className="flex flex-col sm:flex-row items-center gap-6">
        <Skeleton className="h-32 w-32 rounded-full" />
        <div className="space-y-2 text-center sm:text-left">
          <Skeleton className="h-8 w-48" />
          <Skeleton className="h-4 w-32" />
          <Skeleton className="h-6 w-24" />
        </div>
      </div>
      {/* Form Skeleton */}
      <div className="space-y-6">
        <Skeleton className="h-12 w-full" />
        <Skeleton className="h-12 w-full" />
        <Skeleton className="h-12 w-full" />
      </div>
    </div>
  )
}

interface MenuItem {
  icon: React.ComponentType<{ className?: string }>
  label: string
  description: string
  href?: string
  action?: "payment-portal"
}

const MENU_ITEMS: MenuItem[] = [
  {
    icon: Ticket,
    label: "My Bookings",
    description: "View and manage your event bookings",
    href: "/my-bookings",
  },
  {
    icon: CreditCard,
    label: "Payment Methods",
    description: "Manage your saved payment methods",
    action: "payment-portal",
  },
]

export default function ProfilePage() {
  const router = useRouter()
  const { user, isAuthenticated, isLoading: authLoading, logout } = useAuth()
  const [isEditing, setIsEditing] = useState(false)
  const [isSaving, setIsSaving] = useState(false)
  const [isLoadingPortal, setIsLoadingPortal] = useState(false)
  const [formData, setFormData] = useState<ProfileFormData>({
    name: "",
    email: "",
    phone: "",
    location: "",
  })

  useEffect(() => {
    if (!authLoading && !isAuthenticated) {
      router.push("/login?redirect=/profile")
      return
    }

    if (user) {
      setFormData({
        name: user.name || "",
        email: user.email || "",
        phone: "",
        location: "",
      })
    }
  }, [user, isAuthenticated, authLoading, router])

  const handleInputChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const { name, value } = e.target
    setFormData((prev) => ({ ...prev, [name]: value }))
  }

  const handleSave = async () => {
    setIsSaving(true)
    // Simulate API call
    await new Promise((resolve) => setTimeout(resolve, 1000))
    setIsSaving(false)
    setIsEditing(false)
  }

  const handleCancel = () => {
    if (user) {
      setFormData({
        name: user.name || "",
        email: user.email || "",
        phone: "",
        location: "",
      })
    }
    setIsEditing(false)
  }

  const handleLogout = () => {
    logout()
    router.push("/")
  }

  const handlePaymentPortal = async () => {
    if (isLoadingPortal) return

    setIsLoadingPortal(true)
    try {
      const returnUrl = window.location.href
      const response = await paymentApi.createPortalSession(returnUrl)
      window.location.href = response.url
    } catch (error) {
      console.error("Failed to create portal session:", error)
      alert("Failed to open payment settings. Please try again.")
    } finally {
      setIsLoadingPortal(false)
    }
  }

  const handleMenuItemClick = (item: MenuItem) => {
    if (item.action === "payment-portal") {
      handlePaymentPortal()
    } else if (item.href) {
      router.push(item.href)
    }
  }

  const getInitials = (name: string) => {
    return name
      .split(" ")
      .map((n) => n[0])
      .join("")
      .toUpperCase()
      .slice(0, 2)
  }

  if (authLoading) {
    return (
      <main className="min-h-screen bg-background">
        <Header />
        <div className="container mx-auto px-4 lg:px-8 pt-24 pb-16">
          <div className="flex items-center justify-center h-64">
            <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary" />
          </div>
        </div>
      </main>
    )
  }

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
                <User className="h-4 w-4" />
                Account Settings
              </span>
            </div>
            <h1 className="text-4xl lg:text-5xl font-bold text-balance">
              Your{" "}
              <span className="bg-linear-to-r from-primary to-amber-400 bg-clip-text text-transparent">
                Profile
              </span>
            </h1>
            <p className="text-lg text-muted-foreground max-w-xl mx-auto text-pretty">
              Manage your account settings and preferences.
            </p>
          </div>
        </div>
      </section>

      {/* Profile Content */}
      <section className="container mx-auto px-4 lg:px-8 pb-16 lg:pb-24">
        <div className="max-w-4xl mx-auto">
          <div className="grid grid-cols-1 lg:grid-cols-3 gap-8">
            {/* Profile Card */}
            <div className="lg:col-span-2 space-y-6">
              {/* User Info Card */}
              <div className="glass rounded-xl p-6 border border-border/50">
                <div className="flex flex-col sm:flex-row items-center gap-6">
                  {/* Avatar */}
                  <div className="relative group">
                    <div className="h-28 w-28 rounded-full bg-linear-to-br from-primary to-amber-400 flex items-center justify-center text-3xl font-bold text-primary-foreground">
                      {user?.name ? getInitials(user.name) : "U"}
                    </div>
                    <Button
                      size="icon"
                      className="absolute bottom-0 right-0 h-10 w-10 rounded-full bg-primary text-primary-foreground opacity-0 group-hover:opacity-100 transition-opacity shadow-lg"
                    >
                      <Camera className="h-5 w-5" />
                    </Button>
                  </div>

                  {/* User Info */}
                  <div className="flex-1 text-center sm:text-left">
                    <h2 className="text-2xl font-bold text-foreground">{user?.name || "User"}</h2>
                    <p className="text-muted-foreground flex items-center justify-center sm:justify-start gap-2 mt-1">
                      <Mail className="h-4 w-4" />
                      {user?.email || "email@example.com"}
                    </p>
                    <div className="flex items-center justify-center sm:justify-start gap-2 mt-2">
                      <Badge className="bg-primary/20 text-primary border-primary/30">
                        <Shield className="h-3 w-3 mr-1" />
                        {user?.role || "Member"}
                      </Badge>
                      <Badge variant="outline" className="border-border/50">
                        <Calendar className="h-3 w-3 mr-1" />
                        Joined {user?.created_at ? new Date(user.created_at).toLocaleDateString("en-US", { month: "short", year: "numeric" }) : "2025"}
                      </Badge>
                    </div>
                  </div>

                  {/* Edit Button */}
                  {!isEditing && (
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={() => setIsEditing(true)}
                      className="border-primary/50 text-primary hover:bg-primary/10"
                    >
                      <Edit3 className="h-4 w-4 mr-2" />
                      Edit Profile
                    </Button>
                  )}
                </div>
              </div>

              {/* Profile Form */}
              <div className="glass rounded-xl p-6 border border-border/50">
                <h3 className="text-lg font-semibold text-foreground mb-6">Personal Information</h3>
                
                <div className="space-y-6">
                  {/* Name */}
                  <div className="space-y-2">
                    <Label htmlFor="name" className="text-muted-foreground">
                      Full Name
                    </Label>
                    <div className="relative">
                      <User className="absolute left-3 top-1/2 -translate-y-1/2 h-5 w-5 text-muted-foreground" />
                      <Input
                        id="name"
                        name="name"
                        value={formData.name}
                        onChange={handleInputChange}
                        disabled={!isEditing}
                        className="pl-10 glass border-primary/30 focus:border-primary disabled:opacity-70"
                      />
                    </div>
                  </div>

                  {/* Email */}
                  <div className="space-y-2">
                    <Label htmlFor="email" className="text-muted-foreground">
                      Email Address
                    </Label>
                    <div className="relative">
                      <Mail className="absolute left-3 top-1/2 -translate-y-1/2 h-5 w-5 text-muted-foreground" />
                      <Input
                        id="email"
                        name="email"
                        type="email"
                        value={formData.email}
                        onChange={handleInputChange}
                        disabled={true}
                        className="pl-10 glass border-primary/30 focus:border-primary disabled:opacity-70"
                      />
                    </div>
                    <p className="text-xs text-muted-foreground">Email cannot be changed</p>
                  </div>

                  {/* Phone */}
                  <div className="space-y-2">
                    <Label htmlFor="phone" className="text-muted-foreground">
                      Phone Number
                    </Label>
                    <div className="relative">
                      <Phone className="absolute left-3 top-1/2 -translate-y-1/2 h-5 w-5 text-muted-foreground" />
                      <Input
                        id="phone"
                        name="phone"
                        value={formData.phone}
                        onChange={handleInputChange}
                        disabled={!isEditing}
                        placeholder="Add phone number"
                        className="pl-10 glass border-primary/30 focus:border-primary disabled:opacity-70"
                      />
                    </div>
                  </div>

                  {/* Location */}
                  <div className="space-y-2">
                    <Label htmlFor="location" className="text-muted-foreground">
                      Location
                    </Label>
                    <div className="relative">
                      <MapPin className="absolute left-3 top-1/2 -translate-y-1/2 h-5 w-5 text-muted-foreground" />
                      <Input
                        id="location"
                        name="location"
                        value={formData.location}
                        onChange={handleInputChange}
                        disabled={!isEditing}
                        placeholder="Add location"
                        className="pl-10 glass border-primary/30 focus:border-primary disabled:opacity-70"
                      />
                    </div>
                  </div>

                  {/* Action Buttons */}
                  {isEditing && (
                    <div className="flex gap-3 pt-4">
                      <Button
                        onClick={handleSave}
                        disabled={isSaving}
                        className="bg-linear-to-r from-primary to-amber-400 hover:from-amber-400 hover:to-primary text-primary-foreground font-semibold"
                      >
                        {isSaving ? (
                          <>
                            <div className="animate-spin rounded-full h-4 w-4 border-b-2 border-primary-foreground mr-2" />
                            Saving...
                          </>
                        ) : (
                          <>
                            <Check className="h-4 w-4 mr-2" />
                            Save Changes
                          </>
                        )}
                      </Button>
                      <Button
                        variant="outline"
                        onClick={handleCancel}
                        disabled={isSaving}
                        className="border-border/50"
                      >
                        <X className="h-4 w-4 mr-2" />
                        Cancel
                      </Button>
                    </div>
                  )}
                </div>
              </div>
            </div>

            {/* Sidebar Menu */}
            <div className="space-y-6">
              {/* Quick Links */}
              <div className="glass rounded-xl border border-border/50 overflow-hidden">
                <div className="p-4 border-b border-border/50">
                  <h3 className="font-semibold text-foreground flex items-center gap-2">
                    <Settings className="h-4 w-4 text-primary" />
                    Quick Links
                  </h3>
                </div>
                <div className="divide-y divide-border/50">
                  {MENU_ITEMS.map((item) => {
                    const isLoading = item.action === "payment-portal" && isLoadingPortal

                    if (item.href) {
                      return (
                        <Link
                          key={item.label}
                          href={item.href}
                          className="flex items-center gap-4 p-4 hover:bg-primary/5 transition-colors group"
                        >
                          <div className="h-10 w-10 rounded-lg bg-primary/10 flex items-center justify-center text-primary group-hover:bg-primary group-hover:text-primary-foreground transition-colors">
                            <item.icon className="h-5 w-5" />
                          </div>
                          <div className="flex-1">
                            <p className="font-medium text-foreground group-hover:text-primary transition-colors">
                              {item.label}
                            </p>
                            <p className="text-xs text-muted-foreground">{item.description}</p>
                          </div>
                          <ChevronRight className="h-5 w-5 text-muted-foreground group-hover:text-primary transition-colors" />
                        </Link>
                      )
                    }

                    return (
                      <Button
                        key={item.label}
                        variant="ghost"
                        onClick={() => handleMenuItemClick(item)}
                        disabled={isLoading}
                        className="flex items-center gap-4 p-4 h-auto hover:bg-primary/5 transition-colors group w-full justify-start text-left disabled:opacity-50"
                      >
                        <div className="h-10 w-10 rounded-lg bg-primary/10 flex items-center justify-center text-primary group-hover:bg-primary group-hover:text-primary-foreground transition-colors">
                          {isLoading ? (
                            <Loader2 className="h-5 w-5 animate-spin" />
                          ) : (
                            <item.icon className="h-5 w-5" />
                          )}
                        </div>
                        <div className="flex-1">
                          <p className="font-medium text-foreground group-hover:text-primary transition-colors">
                            {isLoading ? "Opening..." : item.label}
                          </p>
                          <p className="text-xs text-muted-foreground">{item.description}</p>
                        </div>
                        <ChevronRight className="h-5 w-5 text-muted-foreground group-hover:text-primary transition-colors" />
                      </Button>
                    )
                  })}
                </div>
              </div>

              {/* Danger Zone */}
              <div className="glass rounded-xl border border-red-500/30 overflow-hidden">
                <div className="p-4 border-b border-red-500/30">
                  <h3 className="font-semibold text-red-400">Danger Zone</h3>
                </div>
                <div className="p-4">
                  <Button
                    variant="outline"
                    onClick={handleLogout}
                    className="w-full border-red-500/50 text-red-400 hover:bg-red-500/10 hover:text-red-400"
                  >
                    <LogOut className="h-4 w-4 mr-2" />
                    Logout
                  </Button>
                </div>
              </div>
            </div>
          </div>
        </div>
      </section>
    </main>
  )
}
