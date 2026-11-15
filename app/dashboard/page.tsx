// Protected /dashboard route: verifies server-side session, GDPR consent, and health profile onboarding
// before rendering the Interactive Health & AI Dashboard.
import GdprConsentCard from "@/feature/auth/consent/GdprConsentCard"
import { HealthDashboardClient } from "@/feature/dashboard/HealthDashboardClient"
import HealthProfileWizard from "@/feature/health-profile/HealthProfileWizard"
import { requireAuthentication } from "@/lib/auth/requireAuthentication"

export const instant = false

export default async function DashboardPage() {
  const session = await requireAuthentication()

  if (!session.consented) {
    return <GdprConsentCard email={session.email} />
  }

  if (!session.hasHealthProfile) {
    return <HealthProfileWizard />
  }

  return <HealthDashboardClient userEmail={session.email} />
}
