import "server-only"
import { NextResponse } from "next/server"

const isProduction = process.env.NODE_ENV === "production"

//set the cookies to the outgoing response from server to browser
export function setAuthCookies(
    response : NextResponse,
    accessToken : string,
    refreshToken : string
) {
    //set the access_token and set the max age to 15 minutes
    response.cookies.set("access_token", accessToken, {
        httpOnly : true,
        secure : isProduction,
        sameSite : "lax",
        path : "/",
        maxAge : 15 * 60
    })

    //set the refresh_token set the max age to 7days
    response.cookies.set("refresh_token", refreshToken, {
        httpOnly : true,
        secure : isProduction,
        sameSite : "lax",
        path : "/api/auth",
        maxAge : 7 * 24 * 60 * 60
    })
}

//clear the cookies
export function clearAuthCookies(response : NextResponse) {
    response.cookies.set("access_token", "", {
        httpOnly : true,
        secure : isProduction,
        sameSite : "lax",
        path : "/",
        maxAge : 0
    })

    response.cookies.set("refresh_token", "", {
        httpOnly : true,
        secure : isProduction,
        sameSite : "lax",
        path : "/api/auth",
        maxAge : 0
    })
}