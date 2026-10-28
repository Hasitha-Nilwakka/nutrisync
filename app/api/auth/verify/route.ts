import { verifyUser } from '@/lib/server/verifyUser'
import { NextRequest, NextResponse } from 'next/server'

//verify the user, the request comes from email verification link
export async function GET(request : NextRequest) {
    //get the token from url parameters
    const token = request.nextUrl.searchParams.get('token')

    //No token or null, return 400
    if (!token) return NextResponse.json({message : 'missing token'}, {status : 400})

    //pass the token to server file
    return await verifyUser(token)
}