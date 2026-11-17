// Protected /profile route: verifies server-side session via requireAuthentication()
// and enforces mandatory GDPR consent before rendering ProfilePageClient.
import GdprConsentCard from "@/feature/auth/consent/GdprConsentCard"
import { ProfilePageClient } from "@/feature/profile/ProfilePageClient"
import { requireAuthentication } from "@/lib/auth/requireAuthentication"

export const instant = false

export default async function ProfilePage() {
  const session = await requireAuthentication()

  if (!session.consented) {
    return <GdprConsentCard email={session.email} />
  }

  return <ProfilePageClient session={session} />
}
