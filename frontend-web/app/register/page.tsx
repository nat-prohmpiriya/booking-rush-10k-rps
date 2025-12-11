"use client"

import type React from "react"
import { useState } from "react"
import Link from "next/link"
import { useRouter } from "next/navigation"
import { Eye, EyeOff } from "lucide-react"
import { Logo } from "@/components/logo"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { SocialLoginButtons } from "@/components/social-login-buttons"
import { useAuth } from "@/contexts/auth-context"

export default function RegisterPage() {
  const router = useRouter()
  const { register, isLoading, error: authError, clearError } = useAuth()
  const [formData, setFormData] = useState({
    name: "",
    email: "",
    password: "",
    confirmPassword: "",
  })
  const [errors, setErrors] = useState<Record<string, string>>({})
  const [success, setSuccess] = useState<Record<string, boolean>>({})
  const [showPassword, setShowPassword] = useState(false)
  const [showConfirmPassword, setShowConfirmPassword] = useState(false)

  const validateField = (name: string, value: string) => {
    switch (name) {
      case "name":
        if (!value) return "Name is required"
        if (value.length < 2) return "Name must be at least 2 characters"
        return ""
      case "email":
        if (!value) return "Email is required"
        if (!/\S+@\S+\.\S+/.test(value)) return "Email is invalid"
        return ""
      case "password":
        if (!value) return "Password is required"
        if (value.length < 8) return "Password must be at least 8 characters"
        if (!/(?=.*[a-z])/.test(value)) return "Password must contain a lowercase letter"
        if (!/(?=.*[A-Z])/.test(value)) return "Password must contain an uppercase letter"
        if (!/(?=.*\d)/.test(value)) return "Password must contain a number"
        if (!/(?=.*[!@#$%^&*(),.?":{}|<>])/.test(value)) return "Password must contain a special character"
        return ""
      case "confirmPassword":
        if (!value) return "Please confirm your password"
        if (value !== formData.password) return "Passwords do not match"
        return ""
      default:
        return ""
    }
  }

  const handleChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const { name, value } = e.target
    setFormData((prev) => ({ ...prev, [name]: value }))
    clearError()

    const error = validateField(name, value)
    setErrors((prev) => ({ ...prev, [name]: error }))
    setSuccess((prev) => ({ ...prev, [name]: !error && value.length > 0 }))

    if (name === "password" && formData.confirmPassword) {
      const confirmError = validateField("confirmPassword", formData.confirmPassword)
      setErrors((prev) => ({ ...prev, confirmPassword: confirmError }))
      setSuccess((prev) => ({ ...prev, confirmPassword: !confirmError }))
    }
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    const newErrors: Record<string, string> = {}
    Object.keys(formData).forEach((key) => {
      const error = validateField(key, formData[key as keyof typeof formData])
      if (error) newErrors[key] = error
    })

    if (Object.keys(newErrors).length === 0) {
      try {
        await register({
          email: formData.email,
          password: formData.password,
          name: formData.name,
        })
        router.push("/")
      } catch {
        // Error is handled in auth context
      }
    } else {
      setErrors(newErrors)
    }
  }

  return (
    <div className="min-h-screen flex items-center justify-center p-4 relative overflow-hidden">
      {/* Background pattern */}
      <div className="absolute inset-0 bg-[radial-gradient(ellipse_at_bottom,_var(--tw-gradient-stops))] from-primary/10 via-background to-background" />
      <div
        className="absolute inset-0 opacity-30"
        style={{
          backgroundImage: `url('/images/auth-bg-2.jpg')`,
          backgroundSize: "cover",
          backgroundPosition: "center",
          filter: "blur(60px)",
        }}
      />

      {/* Register Card */}
      <div className="relative w-full max-w-md">
        <div className="bg-card border border-primary/20 rounded-xl p-8 card-glow backdrop-blur-sm">
          <div className="flex justify-center mb-8">
            <Logo />
          </div>

          <h1 className="text-3xl font-bold text-center mb-2 bg-linear-to-r from-primary via-primary/90 to-primary/70 bg-clip-text text-transparent">
            Create Account
          </h1>
          <p className="text-muted-foreground text-center mb-8">Join us and start booking amazing events</p>

          <form onSubmit={handleSubmit} className="space-y-4">
            {authError && (
              <div className="p-3 text-sm text-destructive bg-destructive/10 border border-destructive/20 rounded-lg">
                {authError}
              </div>
            )}
            <div className="space-y-2">
              <Label htmlFor="name" className="text-foreground">
                Full Name
              </Label>
              <Input
                id="name"
                name="name"
                type="text"
                placeholder="John Doe"
                value={formData.name}
                onChange={handleChange}
                className={`bg-secondary border-border focus:border-primary transition-all ${
                  errors.name ? "border-destructive focus:border-destructive" : ""
                } ${success.name ? "border-success focus:border-success" : ""}`}
              />
              {errors.name && <p className="text-sm text-destructive">{errors.name}</p>}
            </div>

            <div className="space-y-2">
              <Label htmlFor="email" className="text-foreground">
                Email
              </Label>
              <Input
                id="email"
                name="email"
                type="email"
                placeholder="you@example.com"
                value={formData.email}
                onChange={handleChange}
                className={`bg-secondary border-border focus:border-primary transition-all ${
                  errors.email ? "border-destructive focus:border-destructive" : ""
                } ${success.email ? "border-success focus:border-success" : ""}`}
              />
              {errors.email && <p className="text-sm text-destructive">{errors.email}</p>}
            </div>

            <div className="space-y-2">
              <Label htmlFor="password" className="text-foreground">
                Password
              </Label>
              <div className="relative">
                <Input
                  id="password"
                  name="password"
                  type={showPassword ? "text" : "password"}
                  placeholder="••••••••"
                  value={formData.password}
                  onChange={handleChange}
                  className={`bg-secondary border-border focus:border-primary transition-all pr-10 ${
                    errors.password ? "border-destructive focus:border-destructive" : ""
                  } ${success.password ? "border-success focus:border-success" : ""}`}
                />
                <button
                  type="button"
                  onClick={() => setShowPassword(!showPassword)}
                  className="absolute right-3 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground transition-colors"
                >
                  {showPassword ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
                </button>
              </div>
              {errors.password && <p className="text-sm text-destructive">{errors.password}</p>}
              <p className="text-xs text-muted-foreground">
                Min 8 chars, uppercase, lowercase, number, special character
              </p>
            </div>

            <div className="space-y-2">
              <Label htmlFor="confirmPassword" className="text-foreground">
                Confirm Password
              </Label>
              <div className="relative">
                <Input
                  id="confirmPassword"
                  name="confirmPassword"
                  type={showConfirmPassword ? "text" : "password"}
                  placeholder="••••••••"
                  value={formData.confirmPassword}
                  onChange={handleChange}
                  className={`bg-secondary border-border focus:border-primary transition-all pr-10 ${
                    errors.confirmPassword ? "border-destructive focus:border-destructive" : ""
                  } ${success.confirmPassword ? "border-success focus:border-success" : ""}`}
                />
                <button
                  type="button"
                  onClick={() => setShowConfirmPassword(!showConfirmPassword)}
                  className="absolute right-3 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground transition-colors"
                >
                  {showConfirmPassword ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
                </button>
              </div>
              {errors.confirmPassword && <p className="text-sm text-destructive">{errors.confirmPassword}</p>}
            </div>

            <Button
              type="submit"
              disabled={isLoading}
              className="w-full bg-linear-to-r from-primary via-primary/90 to-primary/80 hover:from-primary/90 hover:via-primary/80 hover:to-primary/70 text-primary-foreground font-semibold shadow-lg shadow-primary/20 transition-all disabled:opacity-50"
            >
              {isLoading ? "Creating Account..." : "Create Account"}
            </Button>
          </form>

          <div className="relative my-6">
            <div className="absolute inset-0 flex items-center">
              <div className="w-full border-t border-border" />
            </div>
            <div className="relative flex justify-center text-sm">
              <span className="px-2 bg-card text-muted-foreground">Or continue with</span>
            </div>
          </div>

          <SocialLoginButtons />

          <p className="text-center text-sm text-muted-foreground mt-6">
            Already have an account?{" "}
            <Link href="/login" className="text-primary hover:text-primary/80 font-semibold transition-colors">
              Sign in
            </Link>
          </p>
        </div>
      </div>
    </div>
  )
}
