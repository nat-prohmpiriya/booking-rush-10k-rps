"use client"

import { useState } from "react"
import Link from "next/link"
import { Header } from "@/components/header"
import { Input } from "@/components/ui/input"
import { Button } from "@/components/ui/button"
import {
  Accordion,
  AccordionContent,
  AccordionItem,
  AccordionTrigger,
} from "@/components/ui/accordion"
import {
  HelpCircle,
  Search,
  Ticket,
  CreditCard,
  Calendar,
  Shield,
  RefreshCw,
  Users,
  Mail,
} from "lucide-react"

interface FAQItem {
  question: string
  answer: string
}

interface FAQCategory {
  id: string
  title: string
  icon: React.ElementType
  items: FAQItem[]
}

const FAQ_DATA: FAQCategory[] = [
  {
    id: "booking",
    title: "Booking & Tickets",
    icon: Ticket,
    items: [
      {
        question: "How do I book tickets for an event?",
        answer:
          "To book tickets, simply browse our events, select the event you want to attend, choose your preferred ticket type and quantity, and proceed to checkout. You'll need to create an account or log in to complete your purchase.",
      },
      {
        question: "Can I choose my seats when booking?",
        answer:
          "Seat selection availability depends on the event. Some events offer general admission tickets, while others allow you to choose specific seats. If seat selection is available, you'll see an interactive seating map during the booking process.",
      },
      {
        question: "How many tickets can I buy at once?",
        answer:
          "The maximum number of tickets per transaction varies by event. Most events allow up to 6-8 tickets per order to ensure fair access for all customers. Check the specific event page for exact limits.",
      },
      {
        question: "Will I receive a confirmation after booking?",
        answer:
          "Yes, you'll receive an email confirmation immediately after your booking is complete. This email contains your booking reference number, event details, and e-tickets (if applicable).",
      },
      {
        question: "How do I access my tickets?",
        answer:
          "After purchase, your tickets will be available in your BookingRush account under 'My Bookings'. You can view, download, or display your e-tickets directly from there. We also send tickets to your registered email.",
      },
    ],
  },
  {
    id: "payment",
    title: "Payment & Pricing",
    icon: CreditCard,
    items: [
      {
        question: "What payment methods do you accept?",
        answer:
          "We accept various payment methods including credit/debit cards (Visa, MasterCard, American Express), bank transfers, and digital wallets. Payment options may vary depending on your location.",
      },
      {
        question: "Is my payment information secure?",
        answer:
          "Absolutely. We use industry-standard SSL encryption and comply with PCI DSS standards to protect your payment information. We never store your complete card details on our servers.",
      },
      {
        question: "Are there any additional fees?",
        answer:
          "Ticket prices displayed include the base ticket price. Service fees and booking charges are clearly shown before you complete your purchase. There are no hidden fees.",
      },
      {
        question: "When will I be charged?",
        answer:
          "Your payment is processed immediately when you confirm your booking. For some events with pre-sales, authorization may happen first, with the actual charge closer to the event date.",
      },
      {
        question: "Do you offer installment payments?",
        answer:
          "For select high-value events, we may offer installment payment options through partner services. Check the payment options during checkout to see if this is available for your purchase.",
      },
    ],
  },
  {
    id: "refunds",
    title: "Refunds & Cancellations",
    icon: RefreshCw,
    items: [
      {
        question: "What is your refund policy?",
        answer:
          "Refund policies vary by event and are set by the event organizer. Generally, tickets are non-refundable unless the event is cancelled or rescheduled. Check the specific event terms before purchasing.",
      },
      {
        question: "What happens if an event is cancelled?",
        answer:
          "If an event is cancelled, you'll receive a full refund including service fees. Refunds are typically processed within 7-14 business days to your original payment method.",
      },
      {
        question: "Can I get a refund if I can't attend?",
        answer:
          "Standard tickets are generally non-refundable for personal reasons. However, some events offer ticket insurance or transfer options. Check if your ticket is eligible for resale through our platform.",
      },
      {
        question: "What if an event is postponed?",
        answer:
          "For postponed events, your tickets remain valid for the new date. If you cannot attend the rescheduled event, you may be eligible for a refund during the announced refund window.",
      },
      {
        question: "How long does a refund take?",
        answer:
          "Once approved, refunds typically take 5-10 business days to appear in your account, depending on your bank or payment provider. Credit card refunds may take one billing cycle to appear.",
      },
    ],
  },
  {
    id: "events",
    title: "Events & Venues",
    icon: Calendar,
    items: [
      {
        question: "How do I find events near me?",
        answer:
          "Use our search and filter options to find events by location, date, or category. You can also enable location services to see nearby events automatically.",
      },
      {
        question: "Can I get event notifications?",
        answer:
          "Yes! Create an account and set up notifications for your favorite artists, venues, or event categories. We'll alert you when new events are announced or when tickets go on sale.",
      },
      {
        question: "What should I bring to the event?",
        answer:
          "Typically, you'll need a valid ID and your tickets (digital or printed). Check the specific event page for additional requirements like COVID protocols, prohibited items, or dress codes.",
      },
      {
        question: "Are events accessible for people with disabilities?",
        answer:
          "Most venues offer accessible seating and facilities. Look for accessibility information on the event page, or contact our support team for specific accommodation requests.",
      },
      {
        question: "Can I meet the artists?",
        answer:
          "Some events offer VIP packages that include meet-and-greet opportunities. Check the ticket options for each event to see if premium experiences are available.",
      },
    ],
  },
  {
    id: "account",
    title: "Account & Security",
    icon: Shield,
    items: [
      {
        question: "How do I create an account?",
        answer:
          "Click 'Sign Up' or 'Register' on our website, enter your email and create a password. You can also sign up using your Google or Facebook account for faster registration.",
      },
      {
        question: "I forgot my password. What should I do?",
        answer:
          "Click 'Forgot Password' on the login page and enter your registered email. We'll send you a link to reset your password. The link expires after 24 hours for security.",
      },
      {
        question: "How do I update my account information?",
        answer:
          "Log in to your account and go to 'Profile' or 'Account Settings'. From there, you can update your personal information, contact details, and notification preferences.",
      },
      {
        question: "How do I delete my account?",
        answer:
          "To delete your account, go to Account Settings and select 'Delete Account'. Note that this action is irreversible and you'll lose access to your booking history.",
      },
      {
        question: "Is my personal data protected?",
        answer:
          "Yes, we take data protection seriously. We comply with PDPA and other applicable data protection regulations. Read our Privacy Policy for details on how we handle your information.",
      },
    ],
  },
  {
    id: "groups",
    title: "Groups & Special Requests",
    icon: Users,
    items: [
      {
        question: "Do you offer group discounts?",
        answer:
          "Yes, many events offer group booking discounts for parties of 10 or more. Contact our group sales team at groups@bookingrush.com for special rates and arrangements.",
      },
      {
        question: "Can I book for a corporate event?",
        answer:
          "Absolutely! We offer corporate booking services with dedicated account managers, custom invoicing, and special arrangements. Reach out to our corporate team for more information.",
      },
      {
        question: "Do you offer gift cards?",
        answer:
          "Yes, BookingRush gift cards are available in various denominations. They can be used for any event on our platform and make great gifts for event lovers.",
      },
      {
        question: "Can I transfer tickets to someone else?",
        answer:
          "Ticket transfer policies vary by event. Some tickets can be transferred through your account, while others are non-transferable for security reasons. Check the event terms for details.",
      },
      {
        question: "Do you offer student discounts?",
        answer:
          "Some events offer student pricing. Look for student ticket options on the event page, and be prepared to show valid student ID at the venue.",
      },
    ],
  },
]

function FAQCategorySection({ category }: { category: FAQCategory }) {
  const Icon = category.icon

  return (
    <div id={category.id} className="glass rounded-xl border border-border/50 overflow-hidden scroll-mt-32">
      <div className="p-6 border-b border-border/50 bg-primary/5">
        <div className="flex items-center gap-3">
          <div className="h-10 w-10 rounded-lg bg-primary/20 flex items-center justify-center">
            <Icon className="h-5 w-5 text-primary" />
          </div>
          <h2 className="text-xl font-bold text-foreground">{category.title}</h2>
        </div>
      </div>
      <div className="p-6">
        <Accordion type="single" collapsible className="w-full">
          {category.items.map((item, index) => (
            <AccordionItem key={index} value={`item-${index}`} className="border-border/50">
              <AccordionTrigger className="text-left hover:text-primary hover:no-underline">
                <span className="font-medium text-foreground pr-4">{item.question}</span>
              </AccordionTrigger>
              <AccordionContent>
                <p className="text-muted-foreground leading-relaxed">{item.answer}</p>
              </AccordionContent>
            </AccordionItem>
          ))}
        </Accordion>
      </div>
    </div>
  )
}

export default function FAQPage() {
  const [searchQuery, setSearchQuery] = useState("")
  const [activeCategory, setActiveCategory] = useState<string | null>(null)

  // Filter FAQs based on search
  const filteredCategories = FAQ_DATA.map((category) => ({
    ...category,
    items: category.items.filter(
      (item) =>
        searchQuery === "" ||
        item.question.toLowerCase().includes(searchQuery.toLowerCase()) ||
        item.answer.toLowerCase().includes(searchQuery.toLowerCase())
    ),
  })).filter((category) => category.items.length > 0)

  const totalResults = filteredCategories.reduce((sum, cat) => sum + cat.items.length, 0)

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
                <HelpCircle className="h-4 w-4" />
                Help Center
              </span>
            </div>
            <h1 className="text-4xl lg:text-5xl font-bold text-balance">
              Frequently Asked{" "}
              <span className="bg-linear-to-r from-primary to-amber-400 bg-clip-text text-transparent">
                Questions
              </span>
            </h1>
            <p className="text-lg text-muted-foreground max-w-xl mx-auto text-pretty">
              Find answers to common questions about booking events, payments, refunds, and more.
            </p>

            {/* Search Bar */}
            <div className="max-w-xl mx-auto mt-8">
              <div className="relative">
                <Search className="absolute left-4 top-1/2 -translate-y-1/2 h-5 w-5 text-muted-foreground" />
                <Input
                  type="text"
                  placeholder="Search for answers..."
                  value={searchQuery}
                  onChange={(e) => setSearchQuery(e.target.value)}
                  className="pl-12 pr-4 h-14 text-lg glass border-primary/30 focus:border-primary placeholder:text-muted-foreground/60"
                />
              </div>
              {searchQuery && (
                <p className="text-sm text-muted-foreground mt-2">
                  Found <span className="text-primary font-medium">{totalResults}</span> results
                </p>
              )}
            </div>
          </div>
        </div>
      </section>

      {/* Quick Links */}
      <section className="container mx-auto px-4 lg:px-8 -mt-4 mb-8">
        <div className="flex flex-wrap justify-center gap-2">
          {FAQ_DATA.map((category) => {
            const Icon = category.icon
            return (
              <a
                key={category.id}
                href={`#${category.id}`}
                className="glass px-4 py-2 rounded-full border border-border/50 hover:border-primary/50 hover:bg-primary/5 transition-all flex items-center gap-2 text-sm"
              >
                <Icon className="h-4 w-4 text-primary" />
                <span className="text-foreground">{category.title}</span>
              </a>
            )
          })}
        </div>
      </section>

      {/* FAQ Content */}
      <section className="container mx-auto px-4 lg:px-8 pb-16 lg:pb-24">
        <div className="max-w-4xl mx-auto space-y-8">
          {filteredCategories.length > 0 ? (
            filteredCategories.map((category) => (
              <FAQCategorySection key={category.id} category={category} />
            ))
          ) : (
            <div className="text-center py-16 space-y-4">
              <div className="glass inline-block p-6 rounded-full">
                <Search className="h-12 w-12 text-muted-foreground" />
              </div>
              <h3 className="text-2xl font-semibold text-foreground">No results found</h3>
              <p className="text-muted-foreground max-w-md mx-auto">
                We couldn&apos;t find any questions matching &quot;{searchQuery}&quot;. Try a different search term.
              </p>
              <Button
                onClick={() => setSearchQuery("")}
                className="mt-4 bg-linear-to-r from-primary to-amber-400 text-primary-foreground"
              >
                Clear Search
              </Button>
            </div>
          )}
        </div>

        {/* Still Need Help */}
        <div className="max-w-4xl mx-auto mt-16">
          <div className="glass rounded-xl p-8 border border-primary/30 text-center">
            <div className="inline-block p-4 rounded-full bg-primary/20 mb-4">
              <Mail className="h-8 w-8 text-primary" />
            </div>
            <h3 className="text-2xl font-bold text-foreground mb-2">Still have questions?</h3>
            <p className="text-muted-foreground mb-6 max-w-md mx-auto">
              Can&apos;t find what you&apos;re looking for? Our support team is here to help.
            </p>
            <div className="flex flex-col sm:flex-row gap-4 justify-center">
                <Button className="bg-linear-to-r from-primary to-amber-400 hover:from-amber-400 hover:to-primary text-primary-foreground font-semibold">
                  Contact Support
                </Button>
              <Button variant="outline" className="border-primary/50 text-primary hover:bg-primary/10">
                support@bookingrush.com
              </Button>
            </div>
          </div>
        </div>
      </section>
    </main>
  )
}
