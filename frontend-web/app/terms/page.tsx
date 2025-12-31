"use client"

import { Header } from "@/components/header"
import { FileText, CheckCircle, XCircle, AlertTriangle, Scale, Mail } from "lucide-react"

export default function TermsOfServicePage() {
  return (
    <main data-testid="terms-page" className="min-h-screen bg-background">
      <Header />

      {/* Hero Section */}
      <section data-testid="terms-hero" className="relative pt-24 pb-12 lg:pt-32 lg:pb-16 overflow-hidden">
        {/* Background */}
        <div className="absolute inset-0 bg-[radial-gradient(ellipse_at_top,var(--tw-gradient-stops))] from-primary/20 via-background to-background" />
        <div className="absolute inset-0 bg-[url('/images/grid-pattern.svg')] opacity-5" />

        <div className="container mx-auto px-4 lg:px-8 relative z-10">
          <div className="max-w-3xl mx-auto text-center space-y-6">
            <div className="inline-block glass px-4 py-2 rounded-full">
              <span className="text-primary text-sm font-medium flex items-center gap-2">
                <FileText className="h-4 w-4" />
                Legal
              </span>
            </div>
            <h1 className="text-4xl lg:text-5xl font-bold text-balance">
              Terms of{" "}
              <span className="bg-linear-to-r from-primary to-amber-400 bg-clip-text text-transparent">
                Service
              </span>
            </h1>
            <p className="text-lg text-muted-foreground max-w-xl mx-auto text-pretty">
              Last updated: December 10, 2025
            </p>
          </div>
        </div>
      </section>

      {/* Content */}
      <section data-testid="terms-content" className="container mx-auto px-4 lg:px-8 pb-16 lg:pb-24">
        <div className="max-w-4xl mx-auto space-y-12">
          {/* Introduction */}
          <div className="glass rounded-xl p-8 border border-border/50">
            <p className="text-muted-foreground leading-relaxed">
              Welcome to BookingRush. These Terms of Service (&quot;Terms&quot;) govern your use of our website,
              mobile applications, and services (collectively, the &quot;Services&quot;). By accessing or using our
              Services, you agree to be bound by these Terms. If you do not agree to these Terms, please do not use
              our Services.
            </p>
          </div>

          {/* Acceptance of Terms */}
          <div className="glass rounded-xl border border-border/50 overflow-hidden">
            <div className="p-6 border-b border-border/50 bg-primary/5">
              <div className="flex items-center gap-3">
                <div className="h-10 w-10 rounded-lg bg-primary/20 flex items-center justify-center">
                  <CheckCircle className="h-5 w-5 text-primary" />
                </div>
                <h2 className="text-2xl font-bold text-foreground">Acceptance of Terms</h2>
              </div>
            </div>
            <div className="p-8">
              <p className="text-muted-foreground leading-relaxed mb-4">
                By creating an account, making a purchase, or using any part of our Services, you acknowledge that
                you have read, understood, and agree to be bound by these Terms, as well as our Privacy Policy.
              </p>
              <p className="text-muted-foreground leading-relaxed">
                We reserve the right to modify these Terms at any time. Changes will be effective immediately upon
                posting on our website. Your continued use of the Services after any changes constitutes your
                acceptance of the modified Terms.
              </p>
            </div>
          </div>

          {/* Eligibility */}
          <div className="glass rounded-xl border border-border/50 overflow-hidden">
            <div className="p-6 border-b border-border/50 bg-primary/5">
              <h2 className="text-2xl font-bold text-foreground">Eligibility</h2>
            </div>
            <div className="p-8">
              <p className="text-muted-foreground leading-relaxed mb-4">
                To use our Services, you must:
              </p>
              <ul className="list-disc list-inside space-y-2 text-muted-foreground ml-4">
                <li>Be at least 18 years old or the age of majority in your jurisdiction</li>
                <li>Have the legal capacity to enter into binding contracts</li>
                <li>Not be prohibited from using our Services under applicable laws</li>
                <li>Provide accurate, current, and complete information during registration</li>
                <li>Maintain the security of your account credentials</li>
              </ul>
              <p className="text-muted-foreground leading-relaxed mt-4">
                If you are using our Services on behalf of an organization, you represent that you have the authority
                to bind that organization to these Terms.
              </p>
            </div>
          </div>

          {/* Account Registration */}
          <div className="glass rounded-xl border border-border/50 overflow-hidden">
            <div className="p-6 border-b border-border/50 bg-primary/5">
              <h2 className="text-2xl font-bold text-foreground">Account Registration and Security</h2>
            </div>
            <div className="p-8 space-y-4">
              <p className="text-muted-foreground leading-relaxed">
                To access certain features of our Services, you must create an account. When creating an account, you
                agree to:
              </p>
              <ul className="list-disc list-inside space-y-2 text-muted-foreground ml-4">
                <li>Provide accurate and complete information</li>
                <li>Keep your account information up to date</li>
                <li>Maintain the confidentiality of your password</li>
                <li>Notify us immediately of any unauthorized use of your account</li>
                <li>Take responsibility for all activities that occur under your account</li>
              </ul>
              <p className="text-muted-foreground leading-relaxed">
                You may not transfer your account to another person or use another person&apos;s account without
                permission. We reserve the right to suspend or terminate accounts that violate these Terms.
              </p>
            </div>
          </div>

          {/* Ticket Purchases */}
          <div className="glass rounded-xl border border-border/50 overflow-hidden">
            <div className="p-6 border-b border-border/50 bg-primary/5">
              <h2 className="text-2xl font-bold text-foreground">Ticket Purchases and Bookings</h2>
            </div>
            <div className="p-8 space-y-6">
              <div>
                <h3 className="text-lg font-semibold text-foreground mb-3">Purchase Process</h3>
                <p className="text-muted-foreground leading-relaxed">
                  When you purchase tickets through our platform, you are entering into a contract with the event
                  organizer. We act as an intermediary to facilitate the transaction. All sales are subject to
                  availability and confirmation.
                </p>
              </div>

              <div>
                <h3 className="text-lg font-semibold text-foreground mb-3">Pricing and Fees</h3>
                <ul className="list-disc list-inside space-y-2 text-muted-foreground ml-4">
                  <li>All prices are displayed in Thai Baht (THB) unless otherwise stated</li>
                  <li>Service fees and booking charges are clearly shown before purchase</li>
                  <li>Prices are subject to change without notice until payment is confirmed</li>
                  <li>We reserve the right to correct pricing errors</li>
                </ul>
              </div>

              <div>
                <h3 className="text-lg font-semibold text-foreground mb-3">Payment</h3>
                <p className="text-muted-foreground leading-relaxed">
                  Payment must be made in full at the time of booking. We accept various payment methods as displayed
                  during checkout. By providing payment information, you represent that you are authorized to use the
                  payment method.
                </p>
              </div>

              <div>
                <h3 className="text-lg font-semibold text-foreground mb-3">Order Confirmation</h3>
                <p className="text-muted-foreground leading-relaxed">
                  You will receive an email confirmation upon successful booking. This confirmation serves as proof of
                  purchase. Please check your email (including spam folder) after completing your transaction.
                </p>
              </div>
            </div>
          </div>

          {/* Cancellations and Refunds */}
          <div className="glass rounded-xl border border-border/50 overflow-hidden">
            <div className="p-6 border-b border-border/50 bg-primary/5">
              <h2 className="text-2xl font-bold text-foreground">Cancellations and Refunds</h2>
            </div>
            <div className="p-8 space-y-4">
              <p className="text-muted-foreground leading-relaxed">
                Refund policies vary by event and are set by event organizers. General guidelines include:
              </p>
              <ul className="list-disc list-inside space-y-2 text-muted-foreground ml-4">
                <li>Most tickets are non-refundable unless the event is cancelled or rescheduled</li>
                <li>If an event is cancelled, you will receive a full refund including service fees</li>
                <li>For rescheduled events, your tickets remain valid for the new date</li>
                <li>Refunds are processed within 7-14 business days</li>
                <li>Processing fees may be non-refundable in certain circumstances</li>
              </ul>
              <p className="text-muted-foreground leading-relaxed mt-4">
                Please review the specific refund policy for each event before making a purchase. Once tickets are
                issued, changes to bookings may not be possible.
              </p>
            </div>
          </div>

          {/* Prohibited Activities */}
          <div className="glass rounded-xl border border-border/50 overflow-hidden">
            <div className="p-6 border-b border-border/50 bg-primary/5">
              <div className="flex items-center gap-3">
                <div className="h-10 w-10 rounded-lg bg-red-500/20 flex items-center justify-center">
                  <XCircle className="h-5 w-5 text-red-500" />
                </div>
                <h2 className="text-2xl font-bold text-foreground">Prohibited Activities</h2>
              </div>
            </div>
            <div className="p-8">
              <p className="text-muted-foreground leading-relaxed mb-4">
                You agree not to engage in any of the following prohibited activities:
              </p>
              <ul className="list-disc list-inside space-y-2 text-muted-foreground ml-4">
                <li>Using bots, scripts, or automated tools to purchase tickets</li>
                <li>Reselling tickets for profit above face value (scalping)</li>
                <li>Creating multiple accounts to circumvent purchase limits</li>
                <li>Providing false or misleading information</li>
                <li>Attempting to gain unauthorized access to our systems</li>
                <li>Interfering with the proper functioning of our Services</li>
                <li>Violating any applicable laws or regulations</li>
                <li>Engaging in fraudulent activities</li>
                <li>Harassing, threatening, or abusing other users or our staff</li>
                <li>Using our Services for any illegal or unauthorized purpose</li>
              </ul>
              <p className="text-muted-foreground leading-relaxed mt-4">
                Violation of these prohibitions may result in immediate termination of your account and legal action.
              </p>
            </div>
          </div>

          {/* Intellectual Property */}
          <div className="glass rounded-xl border border-border/50 overflow-hidden">
            <div className="p-6 border-b border-border/50 bg-primary/5">
              <h2 className="text-2xl font-bold text-foreground">Intellectual Property Rights</h2>
            </div>
            <div className="p-8 space-y-4">
              <p className="text-muted-foreground leading-relaxed">
                All content on our platform, including but not limited to text, graphics, logos, images, software, and
                designs, is the property of BookingRush or our licensors and is protected by intellectual property
                laws.
              </p>
              <p className="text-muted-foreground leading-relaxed">
                You may not copy, reproduce, distribute, modify, or create derivative works of any content without our
                express written permission. The BookingRush name and logo are trademarks and may not be used without
                authorization.
              </p>
            </div>
          </div>

          {/* Limitation of Liability */}
          <div className="glass rounded-xl border border-border/50 overflow-hidden">
            <div className="p-6 border-b border-border/50 bg-primary/5">
              <div className="flex items-center gap-3">
                <div className="h-10 w-10 rounded-lg bg-amber-500/20 flex items-center justify-center">
                  <AlertTriangle className="h-5 w-5 text-amber-500" />
                </div>
                <h2 className="text-2xl font-bold text-foreground">Limitation of Liability</h2>
              </div>
            </div>
            <div className="p-8 space-y-4">
              <p className="text-muted-foreground leading-relaxed">
                To the fullest extent permitted by law, BookingRush and its officers, directors, employees, and agents
                shall not be liable for:
              </p>
              <ul className="list-disc list-inside space-y-2 text-muted-foreground ml-4">
                <li>Any indirect, incidental, special, consequential, or punitive damages</li>
                <li>Loss of profits, revenue, data, or use</li>
                <li>Event cancellations, postponements, or changes</li>
                <li>Issues with event venues or organizers</li>
                <li>Technical difficulties or service interruptions</li>
                <li>Unauthorized access to your account due to your negligence</li>
              </ul>
              <p className="text-muted-foreground leading-relaxed">
                Our total liability to you for any claims arising from your use of our Services shall not exceed the
                amount you paid to us in the 12 months preceding the claim.
              </p>
            </div>
          </div>

          {/* Disclaimer of Warranties */}
          <div className="glass rounded-xl border border-border/50 overflow-hidden">
            <div className="p-6 border-b border-border/50 bg-primary/5">
              <h2 className="text-2xl font-bold text-foreground">Disclaimer of Warranties</h2>
            </div>
            <div className="p-8">
              <p className="text-muted-foreground leading-relaxed mb-4">
                Our Services are provided on an &quot;as is&quot; and &quot;as available&quot; basis. We make no
                warranties or representations about:
              </p>
              <ul className="list-disc list-inside space-y-2 text-muted-foreground ml-4">
                <li>The accuracy, reliability, or completeness of event information</li>
                <li>Uninterrupted or error-free operation of our Services</li>
                <li>The quality or nature of events</li>
                <li>Results obtained from using our Services</li>
              </ul>
              <p className="text-muted-foreground leading-relaxed mt-4">
                We disclaim all warranties, express or implied, including warranties of merchantability, fitness for a
                particular purpose, and non-infringement.
              </p>
            </div>
          </div>

          {/* Indemnification */}
          <div className="glass rounded-xl border border-border/50 overflow-hidden">
            <div className="p-6 border-b border-border/50 bg-primary/5">
              <div className="flex items-center gap-3">
                <div className="h-10 w-10 rounded-lg bg-primary/20 flex items-center justify-center">
                  <Scale className="h-5 w-5 text-primary" />
                </div>
                <h2 className="text-2xl font-bold text-foreground">Indemnification</h2>
              </div>
            </div>
            <div className="p-8">
              <p className="text-muted-foreground leading-relaxed">
                You agree to indemnify, defend, and hold harmless BookingRush, its affiliates, officers, directors,
                employees, and agents from any claims, liabilities, damages, losses, costs, or expenses (including
                reasonable attorneys&apos; fees) arising from:
              </p>
              <ul className="list-disc list-inside space-y-2 text-muted-foreground ml-4 mt-4">
                <li>Your use or misuse of our Services</li>
                <li>Your violation of these Terms</li>
                <li>Your violation of any rights of another party</li>
                <li>Your violation of any applicable laws or regulations</li>
              </ul>
            </div>
          </div>

          {/* Governing Law */}
          <div className="glass rounded-xl border border-border/50 overflow-hidden">
            <div className="p-6 border-b border-border/50 bg-primary/5">
              <h2 className="text-2xl font-bold text-foreground">Governing Law and Dispute Resolution</h2>
            </div>
            <div className="p-8 space-y-4">
              <p className="text-muted-foreground leading-relaxed">
                These Terms shall be governed by and construed in accordance with the laws of Thailand, without regard
                to its conflict of law provisions.
              </p>
              <p className="text-muted-foreground leading-relaxed">
                Any disputes arising from these Terms or your use of our Services shall be subject to the exclusive
                jurisdiction of the courts of Bangkok, Thailand.
              </p>
              <p className="text-muted-foreground leading-relaxed">
                We encourage you to contact us first to resolve any disputes informally before pursuing legal action.
              </p>
            </div>
          </div>

          {/* Severability */}
          <div className="glass rounded-xl border border-border/50 overflow-hidden">
            <div className="p-6 border-b border-border/50 bg-primary/5">
              <h2 className="text-2xl font-bold text-foreground">Severability and Waiver</h2>
            </div>
            <div className="p-8 space-y-4">
              <p className="text-muted-foreground leading-relaxed">
                If any provision of these Terms is found to be unenforceable or invalid, that provision shall be
                limited or eliminated to the minimum extent necessary so that the remaining provisions remain in full
                force and effect.
              </p>
              <p className="text-muted-foreground leading-relaxed">
                Our failure to enforce any right or provision of these Terms shall not constitute a waiver of such
                right or provision.
              </p>
            </div>
          </div>

          {/* Contact Information */}
          <div className="glass rounded-xl border border-primary/30 p-8 text-center">
            <div className="inline-block p-4 rounded-full bg-primary/20 mb-4">
              <Mail className="h-8 w-8 text-primary" />
            </div>
            <h3 className="text-2xl font-bold text-foreground mb-2">Questions About These Terms?</h3>
            <p className="text-muted-foreground mb-6 max-w-md mx-auto">
              If you have any questions or concerns about these Terms of Service, please contact us:
            </p>
            <div className="space-y-2 text-muted-foreground">
              <p>
                <strong className="text-foreground">Email:</strong> legal@bookingrush.com
              </p>
              <p>
                <strong className="text-foreground">Address:</strong> 123 Event Street, Bangkok, Thailand 10110
              </p>
              <p>
                <strong className="text-foreground">Phone:</strong> +66 (0) 2-123-4567
              </p>
            </div>
          </div>
        </div>
      </section>
    </main>
  )
}
