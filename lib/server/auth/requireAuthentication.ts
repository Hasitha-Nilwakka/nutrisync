import { cookies } from "next/headers"
import { redirect } from "next/navigation"
import "server-only"
import { verifyAccessToken } from "./tokens"
import prisma from "@/lib/prisma"

//this function verifies the access token when ever a request comes in
//if the tokens are not valid or sessions are expired redirects to login
//if verified it returns the userid
//function is only accessible to server components "server-only" blocks client components
export async function requireAuthentication() {
    const cookieStore = await cookies()
    const accessToken = cookieStore.get("access_token")?.value

    //no access token -> redirect to login
    if (!accessToken) redirect('/')

    let payload

    //verify the token
    try {
        payload = await verifyAccessToken(accessToken)
    } catch {
        redirect('/')
    }

    const session = await prisma.session.findUnique({
        where : {
            id : payload.sid
        },
        select : {
            userId : true,
            expiresAt : true,
            absoluteExpiresAt : true,
            revokedAt : true
        }
    })

    const now = new Date()

    if (
        !session ||
        session.userId !== payload.sub ||
        session.revokedAt !== null ||
        session.expiresAt <= now ||
        session.absoluteExpiresAt <= now
    ) {
        redirect('/')
    }

    return {
        userId : session.userId
    }
}