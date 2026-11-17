// Client container for /profile coordinating account summary, sticky NutriSync header,
// LogoutButton, GdprExportButton, TwoFactorSecurityCard, HealthProfileSettingsForm, and PrivacyAndAccountDeletionCard.
"use client"

import { useState } from "react"
import Link from "next/link"
import { ArrowLeft, Mail, ShieldCheck } from "lucide-react"
import { ThemeToggle } from "@/components/theme/ThemeToggle"
import LogoutButton from "@/components/ui/LogoutButton"
import { GdprExportButton } from "@/feature/dashboard/GdprExportButton"
import { NutriSyncBrand } from "@/feature/landing/NutriSyncBrand"
import { HealthProfileSettingsForm } from "@/feature/profile/HealthProfileSettingsForm"
import { PrivacyAndAccountDeletionCard } from "@/feature/profile/PrivacyAndAccountDeletionCard"
import { TwoFactorSecurityCard } from "@/feature/profile/TwoFactorSecurityCard"
import type { AuthenticatedSession } from "@/lib/auth/requireAuthentication"

interface ProfilePageClientProps {
  session: AuthenticatedSession
}

export function ProfilePageClient({ session }: ProfilePageClientProps) {
  const [twoFactorEnabled, setTwoFactorEnabled] = useState(
    session.twoFactorEnabled
  )

  return (
    <div className="relative min-h-screen bg-background">
      <div
        aria-hidden="true"
        className="pointer-events-none fixed inset-0 -z-10 bg-[radial-gradient(ellipse_at_top,_rgba(11,116,184,0.10),_transparent_55%),radial-gradient(ellipse_at_bottom_right,_rgba(24,169,153,0.08),_transparent_50%)] dark:bg-[radial-gradient(ellipse_at_top,_rgba(47,168,230,0.14),_transparent_55%),radial-gradient(ellipse_at_bottom_right,_rgba(41,208,187,0.10),_transparent_50%)]"
      />

      {/* Sticky Branded Header matching /dashboard */}
      <header className="sticky top-0 z-40 w-full border-b border-border/70 bg-background/85 backdrop-blur-md">
        <div className="mx-auto flex h-16 w-full max-w-5xl items-center justify-between gap-4 px-4 sm:px-6 lg:px-8">
          <Link
            href="/dashboard"
            className="rounded-md focus-visible:outline-2 focus-visible:outline-ring"
          >
            <NutriSyncBrand size="sm" />
          </Link>

          <div className="flex flex-wrap items-center gap-2">
            <Link
              href="/dashboard"
              className="inline-flex h-7 items-center justify-center gap-1.5 rounded-[min(var(--radius-md),12px)] border border-border bg-background px-2.5 text-[0.8rem] font-medium whitespace-nowrap transition-all hover:bg-muted hover:text-foreground"
            >
              <ArrowLeft className="size-4" />
              <span className="hidden sm:inline">Dashboard</span>
            </Link>
            <GdprExportButton />
            <ThemeToggle />
            <LogoutButton />
          </div>
        </div>
      </header>

      <main className="mx-auto flex w-full max-w-5xl flex-col gap-6 px-4 py-8 sm:px-6 lg:px-8">
        {/* Account Header Summary */}
        <section className="flex flex-col justify-between gap-3 rounded-2xl border border-border/80 bg-card p-6 shadow-sm sm:flex-row sm:items-center">
          <div className="space-y-1">
            <div className="flex flex-wrap items-center gap-2">
              <h1 className="text-2xl font-bold tracking-tight">
                Account &amp; Health Profile
              </h1>
              {twoFactorEnabled && (
                <span className="inline-flex items-center gap-1 rounded-full bg-emerald-500/10 px-2.5 py-0.5 text-xs font-medium text-emerald-700 dark:text-emerald-300">
                  <ShieldCheck className="size-3.5" />
                  2FA Protected
                </span>
              )}
            </div>
            <p className="flex items-center gap-1.5 text-sm text-muted-foreground">
              <Mail className="size-4" />
              <span>{session.email}</span>
            </p>
          </div>
        </section>

        {/* TOTP 2FA Security */}
        <TwoFactorSecurityCard
          initialEnabled={twoFactorEnabled}
          onStatusChange={setTwoFactorEnabled}
        />

        {/* Health Profile & Privacy Settings */}
        <HealthProfileSettingsForm
          consented={session.consented}
          hasHealthProfile={session.hasHealthProfile}
        />

        {/* Mandatory GDPR Consent Withdrawal & Account Deletion */}
        <PrivacyAndAccountDeletionCard userEmail={session.email} />
      </main>
    </div>
  )
}
