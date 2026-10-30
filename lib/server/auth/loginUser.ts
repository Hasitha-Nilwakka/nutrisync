import { NextResponse } from "next/server";
import prisma from "../../prisma";
import { LoginFormData } from "../../schemas/loginUserSchema";
import { addNewVerificationAndSend } from "./verifyUser";
import bcrypt from "bcryptjs";
import { createAccessToekn, generateRefreshTokenSecret, hashRefreshTokenSecret } from "./tokens";
import { setAuthCookies } from "./cookies";

const SESSION_LIFETIME_MS = 7 * 24 * 60 * 60 * 1000

export async function loginUser(data : LoginFormData) {
    try {
        const user = await prisma.user.findUnique({
            where : {
                email : data.email
            }
        })
        
        //No user found
        if (!user) return NextResponse.json({message : 'Invalid email or password'}, {status : 401})
        
        //compare password
        const isPasswordCorrect = await bcrypt.compare(data.password, user.password)
        if (!isPasswordCorrect) {
            return NextResponse.json({message : 'Password is incorrect'}, {status : 401})
        }

        //verification token is expired
        //generate and send a new email
        //update the new token and expiry date
        if (user.verificationTokenExpiredAt && user.verificationTokenExpiredAt < new Date()) {
            return await addNewVerificationAndSend(data.email)
        }
        
        //Verification pending
        if (user.verified === false) return NextResponse.json({message : 'Verification pending, please check email'}, {status : 403})
        
        //generate refresh token secret
        const refreshTokenSecret = generateRefreshTokenSecret()
        //get the hashed refresh token secret to be stored in DB
        const refreshTokenHash = hashRefreshTokenSecret(refreshTokenSecret)
        
        //create the session, refresh token record
        //$transaction creates a temporary prisma instant 
        //this ensures that data is passed to db if all goes well
        //otherwise changes are rolled back and throws and error
        const result = await prisma.$transaction(async(tx) => {
            const now = new Date()
            const expiresAt = new Date(now.getTime() + SESSION_LIFETIME_MS)

            //add the session
            const session = await tx.session.create({
                data : {
                    userId : user.id,
                    expiresAt : expiresAt,
                    absoluteExpiresAt : expiresAt
                }
            })

            //add the refresh token
            const refreshTokenRecord = await tx.refreshToken.create({
                data : {
                    sessionId : session.id,
                    tokenHash : refreshTokenHash, 
                    expiresAt : expiresAt
                }
            })

            //create the access token
            const accessToken = await createAccessToekn(user.id, session.id)

            //generate the refresh token combining record id from the db and token secret
            const refreshToken = `${refreshTokenRecord.id}.${refreshTokenSecret}`

            return { accessToken, refreshToken}
        })

        //response for the successfull login
        const response = NextResponse.json({message : "You are logged in"}, {status : 200})

        //attach the cookies to the response
        setAuthCookies(response, result.accessToken, result.refreshToken)

        return response

    } catch(error){
        console.error(error)
        //unable to process
        return NextResponse.json({message : 'Something went wrong, please try again'}, {status : 500})
    }
}