import { NextResponse } from "next/server";
import prisma from "../prisma";

export async function verifyUser(token : string) {
    const user = await prisma.user.findFirst({
        where : {
            verificationToken : token
        }
    })

    const isValid = user?.verificationTokenExpiredAt &&  user?.verificationTokenExpiredAt > new Date(Date.now())

    if (!user || !isValid) {
        return NextResponse.json({message : 'invalid or expired token'}, {status : 400})
    }

    try {
        await prisma.user.update({
            where : {
                id : user.id
            },
            data : {
                verified : true,
                verifiedAt : new Date(Date.now()),
                verificationToken : null,
                verificationTokenExpiredAt : null
            }
        })

        return NextResponse.json({message : 'user verified'}, {status : 200})
    } catch {
        return NextResponse.json({message : 'Verification failed'}, {status : 400})
    }
}