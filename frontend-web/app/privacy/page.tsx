"use client"

import { Header } from "@/components/header"
import { Shield, Eye, Lock, UserCheck, Database, Globe, Mail } from "lucide-react"

export default function PrivacyPolicyPage() {
  return (
    <main data-testid="privacy-page" className="min-h-screen bg-background">
      <Header />

      {/* Hero Section */}
      <section data-testid="privacy-hero" className="relative pt-24 pb-12 lg:pt-32 lg:pb-16 overflow-hidden">
        {/* Background */}
        <div className="absolute inset-0 bg-[radial-gradient(ellipse_at_top,var(--tw-gradient-stops))] from-primary/20 via-background to-background" />
        <div className="absolute inset-0 bg-[url('/images/grid-pattern.svg')] opacity-5" />

        <div className="container mx-auto px-4 lg:px-8 relative z-10">
          <div className="max-w-3xl mx-auto text-center space-y-6">
            <div className="inline-block glass px-4 py-2 rounded-full">
              <span className="text-primary text-sm font-medium flex items-center gap-2">
                <Shield className="h-4 w-4" />
                Legal
              </span>
            </div>
            <h1 className="text-4xl lg:text-5xl font-bold text-balance">
              Privacy{" "}
              <span className="bg-linear-to-r from-primary to-amber-400 bg-clip-text text-transparent">
                Policy
              </span>
            </h1>
            <p className="text-lg text-muted-foreground max-w-xl mx-auto text-pretty">
              Last updated: December 10, 2025
            </p>
          </div>
        </div>
      </section>

      {/* Content */}
      <section data-testid="privacy-content" className="container mx-auto px-4 lg:px-8 pb-16 lg:pb-24">
        <div className="max-w-4xl mx-auto space-y-12">
          {/* Introduction */}
          <div className="glass rounded-xl p-8 border border-border/50">
            <p className="text-muted-foreground leading-relaxed">
              At BookingRush, we take your privacy seriously. This Privacy Policy explains how we collect, use,
              disclose, and safeguard your information when you visit our website and use our services. Please read
              this privacy policy carefully. If you do not agree with the terms of this privacy policy, please do not
              access the site.
            </p>
          </div>

          {/* Information We Collect */}
          <div className="glass rounded-xl border border-border/50 overflow-hidden">
            <div className="p-6 border-b border-border/50 bg-primary/5">
              <div className="flex items-center gap-3">
                <div className="h-10 w-10 rounded-lg bg-primary/20 flex items-center justify-center">
                  <Database className="h-5 w-5 text-primary" />
                </div>
                <h2 className="text-2xl font-bold text-foreground">Information We Collect</h2>
              </div>
            </div>
            <div className="p-8 space-y-6">
              <div>
                <h3 className="text-lg font-semibold text-foreground mb-3">Personal Information</h3>
                <p className="text-muted-foreground leading-relaxed mb-3">
                  We collect information that you provide directly to us when you:
                </p>
                <ul className="list-disc list-inside space-y-2 text-muted-foreground ml-4">
                  <li>Create an account or register for our services</li>
                  <li>Make a purchase or book tickets</li>
                  <li>Subscribe to our newsletter or marketing communications</li>
                  <li>Contact our customer support</li>
                  <li>Participate in surveys, contests, or promotions</li>
                </ul>
                <p className="text-muted-foreground leading-relaxed mt-3">
                  This information may include: name, email address, phone number, billing address, payment
                  information, date of birth, and any other information you choose to provide.
                </p>
              </div>

              <div>
                <h3 className="text-lg font-semibold text-foreground mb-3">Automatically Collected Information</h3>
                <p className="text-muted-foreground leading-relaxed">
                  When you access our website, we automatically collect certain information about your device and
                  usage, including: IP address, browser type, operating system, referring URLs, pages viewed, time
                  spent on pages, links clicked, and other similar information.
                </p>
              </div>

              <div>
                <h3 className="text-lg font-semibold text-foreground mb-3">Cookies and Tracking Technologies</h3>
                <p className="text-muted-foreground leading-relaxed">
                  We use cookies, web beacons, and other tracking technologies to collect information about your
                  browsing behavior and preferences. You can control cookies through your browser settings, but
                  disabling them may affect your ability to use certain features of our site.
                </p>
              </div>
            </div>
          </div>

          {/* How We Use Your Information */}
          <div className="glass rounded-xl border border-border/50 overflow-hidden">
            <div className="p-6 border-b border-border/50 bg-primary/5">
              <div className="flex items-center gap-3">
                <div className="h-10 w-10 rounded-lg bg-primary/20 flex items-center justify-center">
                  <Eye className="h-5 w-5 text-primary" />
                </div>
                <h2 className="text-2xl font-bold text-foreground">How We Use Your Information</h2>
              </div>
            </div>
            <div className="p-8">
              <p className="text-muted-foreground leading-relaxed mb-4">
                We use the information we collect for various purposes, including to:
              </p>
              <ul className="list-disc list-inside space-y-2 text-muted-foreground ml-4">
                <li>Process and fulfill your bookings and transactions</li>
                <li>Create and manage your account</li>
                <li>Send you confirmation emails and booking updates</li>
                <li>Provide customer support and respond to your inquiries</li>
                <li>Send you promotional materials and marketing communications (with your consent)</li>
                <li>Improve our website, services, and user experience</li>
                <li>Detect, prevent, and address fraud and security issues</li>
                <li>Comply with legal obligations and enforce our terms of service</li>
                <li>Analyze usage patterns and trends to enhance our offerings</li>
                <li>Personalize your experience and provide relevant recommendations</li>
              </ul>
            </div>
          </div>

          {/* Information Sharing */}
          <div className="glass rounded-xl border border-border/50 overflow-hidden">
            <div className="p-6 border-b border-border/50 bg-primary/5">
              <div className="flex items-center gap-3">
                <div className="h-10 w-10 rounded-lg bg-primary/20 flex items-center justify-center">
                  <Globe className="h-5 w-5 text-primary" />
                </div>
                <h2 className="text-2xl font-bold text-foreground">Information Sharing and Disclosure</h2>
              </div>
            </div>
            <div className="p-8 space-y-6">
              <p className="text-muted-foreground leading-relaxed">
                We may share your information in the following circumstances:
              </p>

              <div>
                <h3 className="text-lg font-semibold text-foreground mb-3">Service Providers</h3>
                <p className="text-muted-foreground leading-relaxed">
                  We share information with third-party service providers who perform services on our behalf, such as
                  payment processing, data analysis, email delivery, hosting services, and customer service.
                </p>
              </div>

              <div>
                <h3 className="text-lg font-semibold text-foreground mb-3">Event Organizers</h3>
                <p className="text-muted-foreground leading-relaxed">
                  When you purchase tickets, we share necessary information with event organizers to facilitate your
                  attendance and provide you with event-related communications.
                </p>
              </div>

              <div>
                <h3 className="text-lg font-semibold text-foreground mb-3">Legal Requirements</h3>
                <p className="text-muted-foreground leading-relaxed">
                  We may disclose your information if required to do so by law or in response to valid requests by
                  public authorities (e.g., court orders, subpoenas, or government agencies).
                </p>
              </div>

              <div>
                <h3 className="text-lg font-semibold text-foreground mb-3">Business Transfers</h3>
                <p className="text-muted-foreground leading-relaxed">
                  In the event of a merger, acquisition, or sale of assets, your information may be transferred to the
                  acquiring entity.
                </p>
              </div>
            </div>
          </div>

          {/* Data Security */}
          <div className="glass rounded-xl border border-border/50 overflow-hidden">
            <div className="p-6 border-b border-border/50 bg-primary/5">
              <div className="flex items-center gap-3">
                <div className="h-10 w-10 rounded-lg bg-primary/20 flex items-center justify-center">
                  <Lock className="h-5 w-5 text-primary" />
                </div>
                <h2 className="text-2xl font-bold text-foreground">Data Security</h2>
              </div>
            </div>
            <div className="p-8">
              <p className="text-muted-foreground leading-relaxed mb-4">
                We implement appropriate technical and organizational security measures to protect your personal
                information against unauthorized access, alteration, disclosure, or destruction. These measures
                include:
              </p>
              <ul className="list-disc list-inside space-y-2 text-muted-foreground ml-4">
                <li>Encryption of data in transit using SSL/TLS protocols</li>
                <li>Secure payment processing through PCI DSS compliant providers</li>
                <li>Regular security assessments and vulnerability testing</li>
                <li>Access controls and authentication mechanisms</li>
                <li>Employee training on data protection and privacy practices</li>
              </ul>
              <p className="text-muted-foreground leading-relaxed mt-4">
                However, no method of transmission over the Internet or electronic storage is 100% secure. While we
                strive to protect your personal information, we cannot guarantee absolute security.
              </p>
            </div>
          </div>

          {/* Your Rights */}
          <div className="glass rounded-xl border border-border/50 overflow-hidden">
            <div className="p-6 border-b border-border/50 bg-primary/5">
              <div className="flex items-center gap-3">
                <div className="h-10 w-10 rounded-lg bg-primary/20 flex items-center justify-center">
                  <UserCheck className="h-5 w-5 text-primary" />
                </div>
                <h2 className="text-2xl font-bold text-foreground">Your Rights and Choices</h2>
              </div>
            </div>
            <div className="p-8">
              <p className="text-muted-foreground leading-relaxed mb-4">
                Depending on your location, you may have certain rights regarding your personal information,
                including:
              </p>
              <ul className="list-disc list-inside space-y-2 text-muted-foreground ml-4">
                <li>
                  <strong>Access:</strong> Request access to the personal information we hold about you
                </li>
                <li>
                  <strong>Correction:</strong> Request correction of inaccurate or incomplete information
                </li>
                <li>
                  <strong>Deletion:</strong> Request deletion of your personal information
                </li>
                <li>
                  <strong>Portability:</strong> Request a copy of your data in a structured, machine-readable format
                </li>
                <li>
                  <strong>Objection:</strong> Object to the processing of your personal information
                </li>
                <li>
                  <strong>Restriction:</strong> Request restriction of processing your personal information
                </li>
                <li>
                  <strong>Withdraw Consent:</strong> Withdraw consent for marketing communications at any time
                </li>
              </ul>
              <p className="text-muted-foreground leading-relaxed mt-4">
                To exercise these rights, please contact us at privacy@bookingrush.com. We will respond to your
                request within a reasonable timeframe.
              </p>
            </div>
          </div>

          {/* Data Retention */}
          <div className="glass rounded-xl border border-border/50 overflow-hidden">
            <div className="p-6 border-b border-border/50 bg-primary/5">
              <h2 className="text-2xl font-bold text-foreground">Data Retention</h2>
            </div>
            <div className="p-8">
              <p className="text-muted-foreground leading-relaxed">
                We retain your personal information for as long as necessary to fulfill the purposes outlined in this
                Privacy Policy, unless a longer retention period is required or permitted by law. When we no longer
                need your information, we will securely delete or anonymize it.
              </p>
            </div>
          </div>

          {/* Children's Privacy */}
          <div className="glass rounded-xl border border-border/50 overflow-hidden">
            <div className="p-6 border-b border-border/50 bg-primary/5">
              <h2 className="text-2xl font-bold text-foreground">Children&apos;s Privacy</h2>
            </div>
            <div className="p-8">
              <p className="text-muted-foreground leading-relaxed">
                Our services are not intended for children under the age of 13. We do not knowingly collect personal
                information from children under 13. If you are a parent or guardian and believe we have collected
                information from your child, please contact us immediately, and we will take steps to delete such
                information.
              </p>
            </div>
          </div>

          {/* International Transfers */}
          <div className="glass rounded-xl border border-border/50 overflow-hidden">
            <div className="p-6 border-b border-border/50 bg-primary/5">
              <h2 className="text-2xl font-bold text-foreground">International Data Transfers</h2>
            </div>
            <div className="p-8">
              <p className="text-muted-foreground leading-relaxed">
                Your information may be transferred to and processed in countries other than your country of
                residence. These countries may have data protection laws that differ from those in your country. We
                take appropriate safeguards to ensure your personal information remains protected in accordance with
                this Privacy Policy.
              </p>
            </div>
          </div>

          {/* Changes to Policy */}
          <div className="glass rounded-xl border border-border/50 overflow-hidden">
            <div className="p-6 border-b border-border/50 bg-primary/5">
              <h2 className="text-2xl font-bold text-foreground">Changes to This Privacy Policy</h2>
            </div>
            <div className="p-8">
              <p className="text-muted-foreground leading-relaxed">
                We may update this Privacy Policy from time to time to reflect changes in our practices or legal
                requirements. We will notify you of any material changes by posting the new Privacy Policy on this
                page and updating the &quot;Last Updated&quot; date. We encourage you to review this Privacy Policy
                periodically.
              </p>
            </div>
          </div>

          {/* Contact Us */}
          <div className="glass rounded-xl border border-primary/30 p-8 text-center">
            <div className="inline-block p-4 rounded-full bg-primary/20 mb-4">
              <Mail className="h-8 w-8 text-primary" />
            </div>
            <h3 className="text-2xl font-bold text-foreground mb-2">Questions About This Policy?</h3>
            <p className="text-muted-foreground mb-6 max-w-md mx-auto">
              If you have any questions or concerns about this Privacy Policy or our data practices, please contact
              us:
            </p>
            <div className="space-y-2 text-muted-foreground">
              <p>
                <strong className="text-foreground">Email:</strong> privacy@bookingrush.com
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
