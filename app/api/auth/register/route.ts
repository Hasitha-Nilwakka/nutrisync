import { registerFormSchema } from "@/lib/schemas/registerUserSchema"
import { registerNewUser } from "@/lib/server/auth/registerUser"
import { NextResponse } from "next/server"

export async function POST(request : Request) {
    const body = await request.json()
    //validate the data with zod schema
    const parsedData = registerFormSchema.safeParse(body)
    
    //validation failed
    if (!parsedData.success) {
        return NextResponse.json({message : 'Validation failed'}, {status : 400})
    }

    //validation passed: pass the data to server function
    const response = await registerNewUser(parsedData.data)

    return response
}