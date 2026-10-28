import {z} from 'zod'

//Zod schema 
//validates email / confirmEmail, password / confirmPassword and Concent to gdpr
export const registerFormSchema = z.object({
    email : z.email('Invalid email address').min(1, 'Email address is required'),
    password : z.string().trim().min(8, 'Password must be at least 8 characters long'),
    confirmEmail : z.email('Invalid email address').min(1, 'Email address is required'),
    confirmPassword : z.string().trim().min(8, 'Password must be at least 8 characters long'),
    concented : z.boolean().refine(value => value, {error : 'You must agree to the concent to register'})
}) //validate email address and confirm email address
.refine((data) => data.email === data.confirmEmail, {
    message: 'Email addresses do not match',
    path: ['confirmEmail'],
}) //validate password and confirm password
.refine((data) => data.password === data.confirmPassword, {
    message: 'Passwords do not match',
    path: ['confirmPassword']
})

//extrat the type from zod schema
export type RegisterFormData = z.infer<typeof registerFormSchema>