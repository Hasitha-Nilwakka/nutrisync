"use client"
import { Button } from "@/components/ui/button"
import {
  Field,
  FieldDescription,
  FieldError,
  FieldGroup,
  FieldLabel,
  FieldLegend,
  FieldSeparator,
  FieldSet,
} from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import { Spinner } from "@/components/ui/spinner"
import { LoginFormData, loginUserSchema } from "@/lib/schemas/loginUserSchema"
import { zodResolver } from "@hookform/resolvers/zod"
import axios from "axios"
import { useRouter } from "next/navigation"
import { useForm, SubmitHandler } from "react-hook-form"
import { SendPasswordResetVerification } from "../resetpassword/ResetPassword"


export default function LoginForm() {
    const router = useRouter()

    const {register, handleSubmit, reset, formState : {errors, isSubmitting}, setError} = useForm<LoginFormData>({
        resolver : zodResolver(loginUserSchema),
        defaultValues : {
            email : '',
            password : ''
        },
        mode : 'all'
    })

    const onSubmit : SubmitHandler<LoginFormData> = async(data) => {
        try {
            await axios.post(
                '/api/auth/login',
                data
            )
            router.replace("/dashboard")
            router.refresh()
        } catch(error) {
            const message = axios.isAxiosError(error) ? error.response?.data.message : "Login failed, please try again"
            setError('root', {message : message})
        }
    } 
    return (
        <div className="w-full max-w-md">
            <form onSubmit={handleSubmit(onSubmit)}>
                <FieldGroup>
                    <FieldSet>
                        <FieldLegend>Login to your profile</FieldLegend>
                        <FieldDescription>Enter your email & password</FieldDescription>
                        <FieldGroup>
                            <Field>
                                <FieldLabel htmlFor="email">Email</FieldLabel>
                                <Input
                                    id="email"
                                    placeholder="john@example.com"
                                    required
                                    {...register('email')}
                                />
                                {errors.email && <FieldError>{errors.email.message}</FieldError>}
                            </Field>
                            <Field>
                                <FieldLabel htmlFor="password">Password</FieldLabel>
                                <Input
                                    id="password"
                                    type="password"
                                    placeholder="*************"
                                    required
                                    {...register('password')}
                                />
                                {errors.password && <FieldError>{errors.password.message}</FieldError>}
                            </Field>
                            <FieldSeparator/>
                            {errors.root && <FieldError>{errors.root.message}</FieldError>}
                            <Field orientation='horizontal'>
                                <Button type="submit" disabled={isSubmitting}>
                                    {isSubmitting && <Spinner/>}
                                    {isSubmitting ? 'Login in...' : "Login"}
                                </Button>
                                <Button 
                                    variant='outline' 
                                    type='reset' 
                                    disabled={isSubmitting}
                                    onClick={() => reset()}
                                >
                                    Clear
                                </Button>
                            </Field>
                        </FieldGroup>
                    </FieldSet>
                </FieldGroup>
            </form>
            <SendPasswordResetVerification/>
        </div>
    )
}