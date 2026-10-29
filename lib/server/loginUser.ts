import { NextResponse } from "next/server";
import prisma from "../prisma";
import { LoginFormData } from "../schemas/loginUserSchema";
import { addNewVerificationAndSend } from "./verifyUser";
import bcrypt from "bcryptjs";

export async function loginUser(data : LoginFormData) {
    try {
        const user = await prisma.user.findUnique({
            where : {
                email : data.email
            }
        })
        
        //No user found
        if (!user) return NextResponse.json({message : 'Email address not found, please register'}, {status : 404})
        
        //verification token is expired
        //generate and send a new email
        //update the new token and expiry date
        if (user.verificationTokenExpiredAt && user.verificationTokenExpiredAt < new Date()) {
            return await addNewVerificationAndSend(data.email)
        }
        
        //Verification pending
        if (user.verified === false) return NextResponse.json({message : 'Verification pending, please check email'}, {status : 400})
        

        //compare password
        const isPasswordCorrect = await bcrypt.compare(data.password, user.password)
        if (isPasswordCorrect) {
            return NextResponse.json({message : 'You are logged in'}, {status : 200})
        } else {
            return NextResponse.json({message : 'Password is incorrect'}, {status : 401})
        }
    } catch{
        //unable to process
        return NextResponse.json({message : 'Something went wrong, please try again'}, {status : 400})
    }
}