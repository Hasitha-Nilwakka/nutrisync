// Protected /profile route: verifies server-side session via requireAuthentication()
// before rendering ProfilePageClient for 2FA security, health profile settings, and GDPR export.
import { ProfilePageClient } from "@/feature/profile/ProfilePageClient"
import { requireAuthentication } from "@/lib/auth/requireAuthentication"

export const instant = false

export default async function ProfilePage() {
  const session = await requireAuthentication()

  return <ProfilePageClient session={session} />
}
