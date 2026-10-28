import { RegisterFormData } from "../schemas/registerUserSchema";
import bcrypt from 'bcryptjs'
import crypto from 'node:crypto';
import prisma from "../prisma";
import { NextResponse } from "next/server";
import nodemailer from 'nodemailer'
import { google } from 'googleapis';
import dotenv from 'dotenv';

dotenv.config()

//initialize the OAuth client
const OAuth2 = google.auth.OAuth2
const oAuth2Client = new OAuth2(
    process.env.OAUTH_CLIENT_ID,
    process.env.OAUTH_CLIENT_SECRET,
    "https://developers.google.com/oauthplayground"
)

const baseUrl = process.env.APP_URL

//set the refresh token
oAuth2Client.setCredentials({
  refresh_token: process.env.OAUTH_REFRESH_TOKEN,
});

export const registerNewUser = async (data : RegisterFormData) => { 
   const existingUser = await prisma.user.findUnique({
    where : {
        email : data.email
    }
   })

   if (existingUser) return NextResponse.json({message : 'email already registered'}, {status : 409})

   try {
        const {token, expiryDate} = generateVerificationTokenAndDate()
        await prisma.user.create({
            data : {
                email : data.email,
                password : await hashpassword(data.password),
                concented : data.concented,
                verificationToken : token,
                verificationTokenExpiredAt : expiryDate
            }
        })
        await sendVerificationEmail(data.email, token)
        return NextResponse.json({message : 'User created'}, {status : 200})
   } catch {
        return NextResponse.json({message : 'Failed to create user'}, {status : 500})
   }
}

//generate hashed password
export function hashpassword(password : string) {
    return bcrypt.hash(password, 12)
}

//generates a email verification token and generates a expiry date
export function generateVerificationTokenAndDate() {
    const verificationToken = crypto.randomBytes(32).toString('hex')
    const verificationDate : Date = new Date(Date.now() + 24 * 60 * 60 * 1000)

    return {
        token : verificationToken,
        expiryDate : verificationDate
    }
}

//send verification email
export async function sendVerificationEmail(email : string, token : string) {
    const accessTokenResponse = await oAuth2Client.getAccessToken();
    const accessToken = accessTokenResponse?.token;

    if (!accessToken) {
      throw new Error("Failed to retrieve an OAuth access token.");
    }

    const transporter = nodemailer.createTransport({
      service: 'gmail',
      auth: {
        type: 'OAuth2',
        user: process.env.EMAIL_USER,
        clientId: process.env.OAUTH_CLIENT_ID,
        clientSecret: process.env.OAUTH_CLIENT_SECRET,
        refreshToken: process.env.OAUTH_REFRESH_TOKEN,
        accessToken: accessToken,
      },
    });

    const mailOptions = {
        from : "NutriSync user verification",
        to : email,
        subject : 'Please verify your email to complete registration',
        text : `Please click the link to verify`,
        html : `<a href="${baseUrl}/api/auth/verify?token=${token}">Verify Email</a>`
    }

    try {
        const result = await transporter.sendMail(mailOptions);
        console.log(`Verification email sent successfully: ${result.messageId}`);
        return result;
    } catch (error) {
        console.error('Error triggered while sending email:', error);
        throw error;
    }
}