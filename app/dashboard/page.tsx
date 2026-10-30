import LogoutButton from "@/components/ui/LogoutButton";
import { requireAuthentication } from "@/lib/server/auth/requireAuthentication";

//adds a visual pause on the login page untill the JWT is verified and the session is authorized
export const instant = false

export default async function DashboardPage() {
    const { userId } = await requireAuthentication()
    return (
        <div className="text-6xl">
            <p>You are logged in</p>
            <p>{`userId : ${userId}`}</p>
            <LogoutButton/>
        </div>
    )
}