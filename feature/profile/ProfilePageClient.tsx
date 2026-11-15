// Client container for /profile coordinating account summary, navigation back to /dashboard,
// LogoutButton, GdprExportButton, TwoFactorSecurityCard, and HealthProfileSettingsForm.
"use client"

import { useState } from "react"
import Link from "next/link"
import { ArrowLeft, Mail, ShieldCheck } from "lucide-react"
import LogoutButton from "@/components/ui/LogoutButton"
import { Button } from "@/components/ui/button"
import { GdprExportButton } from "@/feature/dashboard/GdprExportButton"
import { HealthProfileSettingsForm } from "@/feature/profile/HealthProfileSettingsForm"
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
    <div className="mx-auto flex min-h-[calc(100dvh-8rem)] w-full max-w-5xl flex-col gap-6 px-4 py-8 sm:px-6 lg:px-8">
      {/* Header & Navigation */}
      <header className="flex flex-col justify-between gap-4 rounded-xl border border-border bg-card p-6 shadow-xs sm:flex-row sm:items-center">
        <div className="space-y-1">
          <div className="flex flex-wrap items-center gap-2">
            <h1 className="text-2xl font-bold tracking-tight">
              Account & Health Profile
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

        <div className="flex flex-wrap items-center gap-2">
          <Button variant="outline" render={<Link href="/dashboard" />}>
            <ArrowLeft className="size-4" />
            Back to Dashboard
          </Button>
          <GdprExportButton />
          <LogoutButton />
        </div>
      </header>

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
    </div>
  )
}
