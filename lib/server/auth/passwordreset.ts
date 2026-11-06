import { generateVerificationTokenAndDate, sendVerificationEmail } from "./generateAndSendVerificationEmails";
import prisma from "@/lib/prisma";
import { addNewVerificationAndSend } from "./verifyUser";
import { generateResponse } from "../response/generateResponse";
import { hashpassword } from "./registerUser";

//send the password reset link
export async function sendPasswordResetLink(email : string) {
    let stage = 'database lookup'
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
        //user not found
        if (!user) {
            return generateResponse.clientError({
                message : 'Email address is not registered',
                title : 'Unauthorized action'
            })
        } 

        //check if the user is verified and verification token is valid
        if (!user.verified) {
            if (user.verificationTokenExpiredAt && user.verificationTokenExpiredAt < new Date()) {
                await addNewVerificationAndSend(email)
                return generateResponse.info({
                    message : 'User account verification pending and expired, new verification send',
                    title : 'User account not verified'
                    })
            }
            return generateResponse.info({
                    message : 'You account is not verified, please verify the account',
                    title : 'Account not verified'
                    })
        }

        //check previous attempts
        stage = 'checking password reset attempts'
        const tooManyAttempts = await tooManyResetAttempts(user.id)
        //if toomany attempts revoke previous verification
        if (tooManyAttempts) {
            return generateResponse.clientError({
                message : 'Too many password reset attempts, please try again later',
                title : 'Action restricted',
                status : 429
            })
        } else {
            stage = 'revoking previous reset tokens'
            await revokeLastPreviousAttempts(user.id)
        }

        //send email
        stage = 'sending reset email'
        await sendVerificationEmail(email, token, 'password')
        
        //save the token
        stage = 'saving reset token'
        await prisma.$transaction([
            prisma.passwordReset.create({
                data : {
                    userId : user.id,
                    verificationToken : token,
                    verificationTokenExpiry : expiryDate
                }
            })
        ])

        return generateResponse.success({
            title : 'Email sent',
            message : 'Verification email sent, please check your email'
        })
    } catch (error) {
        console.error(
            `Password reset failed during ${stage}:`,
            error instanceof Error ? `${error.name}: ${error.message}` : 'Unknown error'
        )
        return generateResponse.internalServerError({
            title : 'Error occured',
            message : 'Failed to send the email, Please try again'
        })
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

//authenticate the user verification request
export async function verifyResetRequest(token : string) {
    if (!token) {
        return generateResponse.internalServerError()
    }

    try {
        //check for matching taoken
        const resetAttempt = await prisma.passwordReset.findFirst({
            where : {
                verificationToken : token
            }
        })

        //token not found
        if (!resetAttempt) {
            return generateResponse.clientError({message : "Unauthorized action"})
        }

        //token expired
        if (resetAttempt.verificationTokenExpiry < new Date()) {
            return generateResponse.clientError({
                message : "Token has expired, please send a new request from the home page", 
                title : "Token expired"
            })
        }

        //token revoked
        if (resetAttempt.revokedAt !== null) {
            return generateResponse.clientError({
                message : "Token has revoked for security reasons, please send a new request from the home page", 
                title : "Token revoked"
            })
        }

        //all checks passed verify the request
        //pass the token to change password UI, changes to db will happen once the actual password change is recorded
        return generateResponse.success({
            title : "Verification successful",
            message : "Redirecting to password change screen",
            data : token
        })
    } catch {
        return generateResponse.internalServerError()
    }
}

//create new password
export async function createNewPassword(token : string, newPassword : string) {
    try {
        const hashedPassword = await hashpassword(newPassword)

        await prisma.$transaction(async(tx) => {
            //get the user id
            const user = await tx.passwordReset.findFirst({
                where : {
                    verificationToken : token
                },
                select : {
                    userId : true,
                    id : true
                }
            })

            //update the password
            await tx.user.update({
                where : {
                    id : user?.userId
                },
                data : {
                    password : hashedPassword
                }
            })

            //update the successful reset attempt
            await tx.passwordReset.update({
                where : {
                    verificationToken : token,
                    id : user?.id
                },
                data : {
                    verified : true
                }
            })

            //revoke all tokens for the user to ensure they are not reused
            await tx.passwordReset.updateMany({
                where : {
                    userId : user?.userId
                },
                data : {
                    revokedAt : new Date()
                }
            })
        })

        return generateResponse.success({
            title : 'Password changed',
            message : 'You can now login, redirecting to login page'
        })
    } catch {
        return generateResponse.internalServerError()
    }
}