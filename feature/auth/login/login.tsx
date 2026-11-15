"use client"
import { useState } from "react"
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
import {
  TwoFactorCodeFormData,
  TwoFactorRequiredLoginResponse,
  twoFactorCodeSchema,
} from "@/lib/schemas/twoFactorSchema"
import { zodResolver } from "@hookform/resolvers/zod"
import axios from "axios"
import { useRouter } from "next/navigation"
import { useForm, SubmitHandler } from "react-hook-form"
import { SendPasswordResetVerification } from "../resetpassword/ResetPassword"


export default function LoginForm() {
    const router = useRouter()
    const [twoFactorTicket, setTwoFactorTicket] = useState<string | null>(null)
    const {register, handleSubmit, reset, formState : {errors, isSubmitting}, setError} = useForm<LoginFormData>({
        resolver : zodResolver(loginUserSchema),
        defaultValues : {
            email : '',
            password : ''
        },
        mode : 'all'
    })

    const twoFactorForm = useForm<TwoFactorCodeFormData>({
        resolver: zodResolver(twoFactorCodeSchema),
        defaultValues: {
            code: '',
        },
        mode: 'onSubmit',
    })

    const onSubmit : SubmitHandler<LoginFormData> = async(data) => {
        try {
            const response = await axios.post<TwoFactorRequiredLoginResponse>(
                '/api/auth/login',
                data,
                { withCredentials: true }
            )
            if (response.data?.require2FA && response.data?.ticket) {
                setTwoFactorTicket(response.data.ticket)
                twoFactorForm.reset({ code: '' })
                return
            }
            router.replace("/dashboard")
            router.refresh()
        } catch(error) {
            const message = axios.isAxiosError(error)
                ? (error.response?.data?.message ?? error.response?.data?.error ?? "Login failed, please try again")
                : "Login failed, please try again"
            setError('root', {message : message})
        }
    }

    const onVerifyTwoFactor: SubmitHandler<TwoFactorCodeFormData> = async (data) => {
        if (!twoFactorTicket) {
            twoFactorForm.setError('root', { message: "2FA session expired. Please log in again." })
            return
        }
        try {
            await axios.post(
                '/api/auth/2fa/login-verify',
                { ticket: twoFactorTicket, code: data.code },
                { withCredentials: true }
            )
            router.replace("/dashboard")
            router.refresh()
        } catch (error) {
            const message = axios.isAxiosError(error)
                ? (error.response?.data?.error ?? error.response?.data?.message ?? "Invalid 2FA code, please try again")
                : "Invalid 2FA code, please try again"
            twoFactorForm.setError('root', { message })
        }
    }

    if (twoFactorTicket) {
        return (
            <div className="w-full max-w-md">
                <form onSubmit={twoFactorForm.handleSubmit(onVerifyTwoFactor)}>
                    <FieldGroup>
                        <FieldSet>
                            <FieldLegend>Two-Factor Authentication</FieldLegend>
                            <FieldDescription>
                                Enter the 6-digit verification code from your authenticator app
                            </FieldDescription>
                            <FieldGroup>
                                <Field>
                                    <FieldLabel htmlFor="two-factor-code">Authenticator Code</FieldLabel>
                                    <Input
                                        id="two-factor-code"
                                        inputMode="numeric"
                                        autoComplete="one-time-code"
                                        maxLength={6}
                                        placeholder="123456"
                                        required
                                        {...twoFactorForm.register('code')}
                                    />
                                    {twoFactorForm.formState.errors.code && (
                                        <FieldError>{twoFactorForm.formState.errors.code.message}</FieldError>
                                    )}
                                </Field>
                                <FieldSeparator />
                                {twoFactorForm.formState.errors.root && (
                                    <FieldError>{twoFactorForm.formState.errors.root.message}</FieldError>
                                )}
                                <Field orientation='horizontal'>
                                    <Button type="submit" disabled={twoFactorForm.formState.isSubmitting}>
                                        {twoFactorForm.formState.isSubmitting && <Spinner />}
                                        {twoFactorForm.formState.isSubmitting ? 'Verifying...' : 'Verify & Login'}
                                    </Button>
                                    <Button
                                        type="button"
                                        variant="outline"
                                        disabled={twoFactorForm.formState.isSubmitting}
                                        onClick={() => {
                                            setTwoFactorTicket(null)
                                            twoFactorForm.reset({ code: '' })
                                        }}
                                    >
                                        Back
                                    </Button>
                                </Field>
                            </FieldGroup>
                        </FieldSet>
                    </FieldGroup>
                </form>
            </div>
        )
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