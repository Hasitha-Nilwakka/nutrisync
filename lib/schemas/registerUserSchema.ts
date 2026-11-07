import { z } from "zod"

// Zod schema for user registration validating email / confirmEmail and password / confirmPassword.
// GDPR consent is collected after authentication via the unified GdprConsentCard on /dashboard.
export const registerFormSchema = z
  .object({
    email: z.email("Invalid email address").min(1, "Email address is required"),
    password: z.string().trim().min(8, "Password must be at least 8 characters long"),
    confirmEmail: z.email("Invalid email address").min(1, "Email address is required"),
    confirmPassword: z.string().trim().min(8, "Password must be at least 8 characters long"),
  })
  .refine((data) => data.email === data.confirmEmail, {
    message: "Email addresses do not match",
    path: ["confirmEmail"],
  })
  .refine((data) => data.password === data.confirmPassword, {
    message: "Passwords do not match",
    path: ["confirmPassword"],
  })

export type RegisterFormData = z.infer<typeof registerFormSchema>
