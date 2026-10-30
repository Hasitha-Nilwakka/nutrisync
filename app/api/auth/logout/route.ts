import prisma from "@/lib/prisma";
import { clearAuthCookies } from "@/lib/server/auth/cookies";
import { cookies } from "next/headers";
import { NextResponse } from "next/server";

//logs out the user
//revoke the session and refresh token
//clears the cookie

export async function POST() {
    try {
        const cookieStore = await cookies();
        const fullRefreshToken = cookieStore.get("refresh_token")?.value;

        if (fullRefreshToken) {
            const separator = fullRefreshToken.indexOf(".");

            if (separator > 0 && separator < fullRefreshToken.length - 1) {
                const tokenId = fullRefreshToken.slice(0, separator);

                const storedToken = await prisma.refreshToken.findUnique({
                    where: { id: tokenId },
                    select: { sessionId: true }
                });

                if (storedToken) {
                    const now = new Date();

                    await prisma.$transaction([
                        prisma.session.updateMany({
                            where: {
                                id: storedToken.sessionId,
                                revokedAt: null
                            },
                            data: { revokedAt: now }
                        }),

                        prisma.refreshToken.updateMany({
                            where: {
                                sessionId: storedToken.sessionId,
                                revokedAt: null
                            },
                            data: { revokedAt: now }
                        })
                    ]);
                }
            }
        }

        const response = NextResponse.json(
            { message: "Logged out successfully" },
            { status: 200 }
        );

        clearAuthCookies(response);

        return response;

    } catch (error) {
        console.error("Failed to log out:", error);

        const response = NextResponse.json(
            { message: "Logged out locally, but server-side revocation may have failed." },
            { status: 500 }
        );
        
        clearAuthCookies(response);

        return response;
    }
}
