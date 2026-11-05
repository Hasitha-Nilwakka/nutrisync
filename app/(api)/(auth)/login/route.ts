import { loginUserSchema } from "@/lib/schemas/loginUserSchema"
import { loginUser } from "@/lib/server/auth/loginUser"
import { NextResponse } from "next/server"

export async function POST(request : Request) {

    try {
        const data = await request.json()
    
        const parsedData = loginUserSchema.safeParse(data)
    
        if (!parsedData.success) {
            return NextResponse.json({message : 'data validation failed'}, {status : 400})
        }
    
        const response = await loginUser(parsedData.data)
    
        return response
    } catch {
        return NextResponse.json(
      { message: "Invalid request" },
      { status: 400 },
    )
    }
}