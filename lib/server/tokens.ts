import "server-only"
import {type JWTPayload, SignJWT, jwtVerify} from "jose"
import { createHash, randomBytes } from "crypto"

//short lived token, limit to fifteen minutes
const ACCESS_TOKEN_LIFETIME_SECONDS = 15 * 60
const ACCESS_TOKEN_ISSUER = "NUTRISYNC"
const ACCESS_TOKEN_AUDIENCE = "NUTRISYNC-USERS"

//get the secret key and convert to raw binary
function getAccessTokenSecret() : Uint8Array {
    const secret = process.env.ACCESS_TOKEN_SECRET

    if (!secret) {
        throw new Error("ACCESS_TOKEN_SECRET is npot configured in .env")
    }

    const key = Buffer.from(secret, "base64")

    //check the length of the key
    if(key.length < 32) {
        throw new Error("ACCESS_TOKEN_SECRET must decode to at least 32 bytes")
    }

    return new Uint8Array(key)
}

export type AccessTokenPayload = JWTPayload & {
    sub : string //subject (userId)
    sid : string //session id
}

//create the token returns a string , inject the userId and sessionId
//set the ACCESS_TOKEN_SECRET as the signature of the token
export async function createAccessToekn(userId : string, sessionId : string) : Promise<string> {
    return new SignJWT({sid : sessionId})
        .setProtectedHeader({alg : "HS256", typ : "JWT"})
        .setSubject(userId)
        .setIssuer(ACCESS_TOKEN_ISSUER)
        .setAudience(ACCESS_TOKEN_AUDIENCE)
        .setIssuedAt()
        .setExpirationTime(`${ACCESS_TOKEN_LIFETIME_SECONDS}s`)
        .sign(getAccessTokenSecret())
}

//Verify the access token with the secret stored in .env
//JwtVerify throw and error in the case of tampered token
export async function verifyAccessToken(token : string) : Promise<AccessTokenPayload> {
    const { payload } = await jwtVerify(
        token,
        getAccessTokenSecret(),
        {
            algorithms : ["HS256"],
            issuer : ACCESS_TOKEN_ISSUER,
            audience : ACCESS_TOKEN_AUDIENCE,
        }
    )

    //verify the type of sub & sid as a additional check
    if (typeof payload.sub !== 'string' || typeof payload.sid !== 'string') {
        throw new Error("Invalid access token claims")
    }

    return payload as AccessTokenPayload
}

//generate a new Refresh Token Secret
export function generateRefreshTokenSecret() : string {
    return randomBytes(32).toString("base64url")
}

//hash the refresh token before stroing in the database
export function hashRefreshTokenSecret(secret : string) : string {
    return createHash("sha256").update(secret, "utf8").digest("hex")
}