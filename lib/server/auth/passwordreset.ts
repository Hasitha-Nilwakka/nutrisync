"use server"
import { NextResponse } from "next/server";
import { generateVerificationTokenAndDate, sendVerificationEmail } from "./generateAndSendVerificationEmails";
import prisma from "@/lib/prisma";
import { addNewVerificationAndSend } from "./verifyUser";

export async function sendPasswordResetLink(email : string) {
    try {
        const {token, expiryDate} = generateVerificationTokenAndDate('password')
        //check the user exists
        const user = await prisma.user.findUnique({
            where : {
                email : email
            },
            select : {
                id : true,
                verified : true,
                verificationTokenExpiredAt : true
            }
        })
        if (!user) return NextResponse.json({message : 'Email address is not registered, please check again'}, {status : 401})
        //check if the user is verified and verification token is valid
        if (!user.verified) {
            if (user.verificationTokenExpiredAt && user.verificationTokenExpiredAt < new Date()) {
                await addNewVerificationAndSend(email)
                return NextResponse.json({message : 'User account verification pending and expired, new verification send'}, {status : 400})    
            }
            return NextResponse.json({message : 'User account verification pending'}, {status : 400})
        }

        //check previous attempts
        const tooManyAttempts = await tooManyResetAttempts(user.id)
        //if toomany attempts revoke previous verification
        if (tooManyAttempts) {
            revokeLastPreviousAttempts(user.id)
        }
        //save the token
        await prisma.$transaction([
            prisma.passwordReset.create({
                data : {
                    userId : user.id,
                    verificationToken : token,
                    verificationTokenExpiry : expiryDate
                }
            })
        ])
        //send email
        await sendVerificationEmail(email, token, 'password')

        return NextResponse.json({message :"Verification send"}, {status : 200})
    } catch {
        return NextResponse.json({message : 'Failed to send verification'}, {status : 500})
    }
}


//check if the user has attempted to reset the password 5 times in last 24 hours
async function tooManyResetAttempts(userId : string) {
    const twentyFourHoursAgo = new Date(Date.now() - 24 * 60 * 60 * 1000)
    const countLast24Hours = await prisma.passwordReset.count({
        where : {
            userId : userId,
            createdAt : {
                gte : twentyFourHoursAgo
            }
        }
    })

    return countLast24Hours >= 5
}

//invalidate last five reset verifications
async function revokeLastPreviousAttempts(userId : string) {
    await prisma.passwordReset.updateMany({
        where : {
            userId : userId
        },
        data : {
            revokedAt : new Date
        }
    })
}

//Verification flow
//check the validity, and its not revoked
//if valid validate -> mark veryfied true, revoke move to reset screen
//if not ask to send new verification