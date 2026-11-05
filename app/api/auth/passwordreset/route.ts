//POST -> verify token

import { sendPasswordResetLink } from "@/lib/server/auth/passwordreset";
import { NextRequest, NextResponse } from "next/server";


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
        return NextResponse.json({message : 'something went wrong'}, {status : 400})
    }
}

//password verification, get the token from URL
export async function GET() {

}