//POST -> verify token

import { sendPasswordResetLink, verifyResetRequest } from "@/lib/server/auth/passwordreset";
import { generateResponse } from "@/lib/server/response/generateResponse";
import { NextRequest } from "next/server";
import { redirect } from "next/navigation";


//called in sending verification link and actual password change
export async function POST(request : NextRequest) {
    try {
        const body = await request.json()
        const {emailReset} = body

        //sending the email verification
        if (emailReset) {
            const response = await sendPasswordResetLink(emailReset)
            return response
        }
    } catch {
        return generateResponse.internalServerError()
    }
}

//password verification, get the token from URL
export async function GET(request : NextRequest) {
    try {
        const token = request.nextUrl.searchParams.get('token')
        if (!token) return generateResponse.clientError({message : "Token is required"})
        const response = await verifyResetRequest(token)
        
        if (response.status === 200) {
            redirect(`/new-password?token=${token}`)
        }
    } catch {
        return generateResponse.internalServerError()
    }
}