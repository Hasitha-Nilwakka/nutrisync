import prisma from "@/lib/prisma"
import { clearAuthCookies, setAuthCookies } from "@/lib/server/auth/cookies"
import { createAccessToekn, generateRefreshTokenSecret, hashRefreshTokenSecret } from "@/lib/server/auth/tokens"
import { cookies } from "next/headers"
import { NextResponse } from "next/server"
import { timingSafeEqual } from "node:crypto"

//this route issues a new access token and rotate the refreshtoken 

//match two hashes
function matchHashes(a : string, b : string) : boolean{
    const left = Buffer.from(a, "hex")
    const right = Buffer.from(b, "hex")

    //time safe equal prevents timing attacks
    //it ensures that it takes the same exact time to process equal or non-equal comparisions
    //this prevents hackers from time gussing combinations
    return (
        left.length === right.length &&
        left.length > 0 && 
        timingSafeEqual(left, right)
    )
}

//request the user to login again
function unauthorized() {
    const response = NextResponse.json(
        {message : 'Authentication required, please login'},
        {status : 401}
    )

    clearAuthCookies(response)

    return response
}

export async function POST() {
    try {
    const cookieStore = await cookies()
    const fullToken = cookieStore.get("refresh_token")?.value

    if (!fullToken) return unauthorized()

    //evaluate the token format
    //expect the format dbstoredrefreshtokenid.refreshtokensecret
    const tokenSeperatorIndex = fullToken.indexOf(".")

    //check for scenarios that break the expected format
    //index < 0 = no dot found
    //index === 0 = dot is at the exact start 
    //index === tokenlength -1 = dot at the exact end
    if (tokenSeperatorIndex <= 0 || tokenSeperatorIndex === fullToken.length -1) {
        return unauthorized()
    }
    //seperate the tokenId and secret
    const tokenId = fullToken.slice(0, tokenSeperatorIndex)
    const secret = fullToken.slice(tokenSeperatorIndex + 1)

    //get the session for the token id and also retrive the associated session for the token
    const storedToken = await prisma.refreshToken.findUnique({
        where : {
            id : tokenId
        }, 
        include : {
            session : true
        }
    })

    //unrecognized token, ask to login
    if (!storedToken) return unauthorized()

    //refresh token secret was hashed before saving into db
    //we need to hash the secret from the cookie before comparing
    const cookieTokenHash = hashRefreshTokenSecret(secret)

    //validate the cookie token matches db stored token hash
    if(!matchHashes(cookieTokenHash, storedToken.tokenHash)) {
        return unauthorized()
    }

    const now = new Date()

    //we mark the previous used refresh tokens with usedAt date
    //so if this tokens usedAt is not equal to null then it could be a stolen refresh token
    //we clear the session for enhanced security and force the user to login to prevent security issues
    //also marked all the non revoked sessions and non revoked associated refresh tokens as revoked
    if (storedToken.usedAt !== null) {
        await prisma.$transaction([
            prisma.session.updateMany({
                where : {
                    id : storedToken.sessionId,
                    revokedAt : null
                },
                data : {revokedAt : now}
            }),
            prisma.refreshToken.updateMany({
                where : {
                    sessionId : storedToken.sessionId,
                    revokedAt : null
                },
                data : {revokedAt : now}
            })
        ])

        return unauthorized()
    }

    const session = storedToken.session

    //validate the session and tokenlifetime and revocation status
    if (
        storedToken.revokedAt !== null ||
        storedToken.expiresAt <= now ||
        session.revokedAt !== null ||
        session.expiresAt <= now ||
        session.absoluteExpiresAt <= now
    ) {
        return unauthorized()
    }

    //prepare new credentials
    const newSecret = generateRefreshTokenSecret()
    const newSecretHash = hashRefreshTokenSecret(newSecret)

    //create new access token
    const newAccessToken = await createAccessToekn(session.userId, session.id)

    const rotation = await prisma.$transaction(async(tx) => {
        //update the current refresh token as used
        const consumed = await tx.refreshToken.updateMany({
            where : {
                id : storedToken.id,
                tokenHash : storedToken.tokenHash,
                revokedAt : null,
                usedAt : null,
                expiresAt : {gt : now}
            },
            data : {usedAt : now}
        })

        //check if the token is consumed by only one request otherwise return false
        if (consumed.count !== 1) return {success : false as const }

        //ensure the session is active
        const activeSession = await tx.session.findFirst({
            where : {
                id : session.id,
                userId : session.userId,
                revokedAt: null,
                expiresAt : {gt : now},
                absoluteExpiresAt : { gt : now}
            },
            select : {id : true}
        })

        if (!activeSession) {
            throw new Error("Session is no longer active")
        }

        //add the new refresh token
        const newRecord = await tx.refreshToken.create({
            data : {
                sessionId : session.id,
                tokenHash : newSecretHash,
                expiresAt : session.expiresAt
            }
        })

        //finally return the refresh token
        return {
            success : true as const,
            refreshToken : `${newRecord.id}.${newSecret}`
        }
    })

    //Another request has consumed the token -> revoke the session
    if (!rotation.success) {
        await prisma.$transaction([
            prisma.session.updateMany({
                where : {
                    id : session.id,
                    revokedAt : null
                },
                data : {revokedAt : new Date(), lastRotatedAt : now}
            }),
            prisma.refreshToken.updateMany({
                where : {
                    sessionId : session.id,
                    revokedAt : null
                },
                data : {revokedAt : new Date()}
            })
        ])

        return unauthorized()
    }

    const response = NextResponse.json({message : 'session refreshed'}, {status : 200})
    //set new auth cookies
    setAuthCookies(response, newAccessToken, rotation.refreshToken)

    return response
    } catch (error) {
        console.error("Refresh failed", error)

        const response = NextResponse.json({message : "Refresh failed, please login again"}, {status : 500})

        clearAuthCookies(response)

        return response
    }
}