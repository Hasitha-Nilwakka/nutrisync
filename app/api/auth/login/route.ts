import { loginUserSchema } from "@/lib/schemas/loginUserSchema"
import { loginUser } from "@/lib/server/loginUser"
import { NextResponse } from "next/server"

export async function POST(request : Request) {
    const data = await request.json()

    const parsedData = loginUserSchema.safeParse(data)

    if (!parsedData.success) {
        return NextResponse.json({message : 'data validation failed'}, {status : 400})
    }

    const response = await loginUser(data)

    return response
}