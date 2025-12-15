"use client"

import Link from "next/link"
import { usePathname } from "next/navigation"

export function Footer() {
  const pathname = usePathname()

  // Hide footer on organizer pages (has its own layout)
  if (pathname.startsWith("/organizer")) {
    return null
  }

  return (
    <footer className="bg-gold-gradient uppercase">
      <div className="container mx-auto px-4 lg:px-8 py-12">
        <div className="grid grid-cols-1 md:grid-cols-4 gap-8">
          <div className="space-y-4">
            <div className="text-2xl font-bold text-black">
              BookingRush
            </div>
            <p className="text-sm text-black font-bold">
              Your premier destination for luxury event booking experiences.
            </p>
          </div>
          <div>
            <h3 className="font-bold mb-4 text-black">Company</h3>
            <ul className="space-y-2 text-sm text-black font-bold">
              <li>
                <Link href="/about" className="hover:text-gray-700 transition-colors">
                  About Us
                </Link>
              </li>
            </ul>
          </div>
          <div>
            <h3 className="font-bold mb-4 text-black">Support</h3>
            <ul className="space-y-2 text-sm text-black font-bold">
              <li>
                <a href="https://n2p.painaina.com/" target="_blank" rel="noopener noreferrer" className="hover:text-gray-700 transition-colors">
                  Contact Us
                </a>
              </li>
              <li>
                <Link href="/faq" className="hover:text-gray-700 transition-colors">
                  FAQ
                </Link>
              </li>
            </ul>
          </div>
          <div>
            <h3 className="font-bold mb-4 text-black">Legal</h3>
            <ul className="space-y-2 text-sm text-black font-bold">
              <li>
                <Link href="/privacy" className="hover:text-gray-700 transition-colors">
                  Privacy Policy
                </Link>
              </li>
              <li>
                <Link href="/terms" className="hover:text-gray-700 transition-colors">
                  Terms of Service
                </Link>
              </li>
            </ul>
          </div>
        </div>
        <div className="mt-12 pt-8 border-t border-black/20 text-center text-sm text-black font-bold">
          <p>© 2025 BookingRush. All rights reserved.</p>
        </div>
      </div>
    </footer>
  )
}
