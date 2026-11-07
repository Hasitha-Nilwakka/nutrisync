import LogoutButton from "@/components/ui/LogoutButton";
import GdprConsentCard from "@/feature/auth/consent/GdprConsentCard";
import HealthProfileWizard from "@/feature/health-profile/HealthProfileWizard";
import { requireAuthentication } from "@/lib/auth/requireAuthentication";

//adds a visual pause on the login page untill the JWT is verified and the session is authorized
export const instant = false

export default async function DashboardPage() {
    const session = await requireAuthentication()

    if (!session.consented) {
        return <GdprConsentCard email={session.email} />
    }

    if (!session.hasHealthProfile) {
        return <HealthProfileWizard />
    }

    return (
        <div className="text-6xl">
            <p>You are logged in</p>
            <p>{`userId : ${session.userId}`}</p>
            <LogoutButton/>
        </div>
    )
}
