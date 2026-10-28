import {z} from 'zod'

//zod schema for user login
export const loginUserSchema = z.object({
    email : z.email('Invalid email').min(1, 'Email address is required'),
    password : z.string().trim().min(8, 'Password must be at least 8 characters long')
})

//extract the type directly from zod
export type LoginFormData = z.infer<typeof loginUserSchema>