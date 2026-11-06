import {z} from 'zod'

//Zod schema 
//validates new password / confirmEmail, password
export const passwordChangeSchema = z.object({
    password : z.string().trim().min(8, 'Password must be at least 8 characters long'),
    confirmPassword : z.string().trim().min(8, 'Password must be at least 8 characters long')
}) 
//validate password and confirm password
.refine((data) => data.password === data.confirmPassword, {
    message: 'Passwords do not match',
    path: ['confirmPassword']
})

//extrat the type from zod schema
export type PasswordChangeFormData = z.infer<typeof passwordChangeSchema>