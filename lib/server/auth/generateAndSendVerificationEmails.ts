import crypto from 'node:crypto';
import nodemailer from 'nodemailer'
import { google } from 'googleapis';
import dotenv from 'dotenv';

//Common function to generate a verification token and send
//Called for user verification and password reset verification

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

type tokenType = 'user' | 'password'

//generate a new verification token
export function generateVerificationTokenAndDate(tokenType : tokenType) {
    const timeMs = tokenType === 'user' ? 24 * 60 * 60 * 1000 : 30 * 60 * 1000
    const verificationToken = crypto.randomBytes(32).toString('hex')
    const verificationDate : Date = new Date(Date.now() + timeMs)

    return {
        token : verificationToken,
        expiryDate : verificationDate
    }
}

//send the verification email
export async function sendVerificationEmail(email : string, token : string, tokenType : tokenType) {
    /*const accessTokenResponse = await oAuth2Client.getAccessToken();
    const accessToken = accessTokenResponse?.token;

    if (!accessToken) {
      throw new Error("Failed to retrieve an OAuth access token.");
    }*/

    const transporter = nodemailer.createTransport({
      service: 'gmail',
      auth: {
        user: process.env.EMAIL_USER,
        pass : process.env.EMAIL_APP_PASSWORD,
        /*clientId: process.env.OAUTH_CLIENT_ID,
        clientSecret: process.env.OAUTH_CLIENT_SECRET,
        refreshToken: process.env.OAUTH_REFRESH_TOKEN,
        accessToken: accessToken,*/
      },
    });

    const userVerificationEmail = {
        from : "NutriSync user verification",
        to : email,
        subject : 'Please verify your email to complete registration',
        text : `Please click the link to verify`,
        html : `<a href="${baseUrl}/api/auth/verify?token=${token}">Verify Email</a>`
    }

    const passwordResetEmail = {
        from : "NutriSync password reset verification",
        to : email,
        subject : 'Please click the link below to complete your password reset',
        text : `Please click the link`,
        html : `<a href="${baseUrl}/api/auth/passwordreset?token=${token}">Reset Password</a>`
    }

    const mailOptions = tokenType === 'user' ? userVerificationEmail : passwordResetEmail

    try {
        const result = await transporter.sendMail(mailOptions);
        console.log(`Verification email sent successfully: ${result.messageId}`);
        return result;
    } catch (error) {
        console.error('Error triggered while sending email:', error);
        throw error;
    }
}