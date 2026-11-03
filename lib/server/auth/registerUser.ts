import { RegisterFormData } from "../../schemas/registerUserSchema";
import bcrypt from 'bcryptjs'
import prisma from "../../prisma";
import { NextResponse } from "next/server";
import { generateVerificationTokenAndDate, sendVerificationEmail } from "./generateAndSendVerificationEmails";

export const registerNewUser = async (data : RegisterFormData) => { 
   const existingUser = await prisma.user.findUnique({
    where : {
        email : data.email
    }
   })

   if (existingUser) return NextResponse.json({message : 'email already registered'}, {status : 409})

   try {
        const {token, expiryDate} = generateVerificationTokenAndDate('user')
        await prisma.user.create({
            data : {
                email : data.email,
                password : await hashpassword(data.password),
                concented : data.concented,
                verificationToken : token,
                verificationTokenExpiredAt : expiryDate
            }
        })
        await sendVerificationEmail(data.email, token, 'user')
        return NextResponse.json({message : 'User created'}, {status : 200})
   } catch {
        return NextResponse.json({message : 'Failed to create user'}, {status : 500})
   }
}

//generate hashed password
export function hashpassword(password : string) {
    return bcrypt.hash(password, 12)
}