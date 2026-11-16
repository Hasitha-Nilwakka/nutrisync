"use client"
import { Button } from "@/components/ui/button"
import {
  Field,
  FieldDescription,
  FieldError,
  FieldGroup,
  FieldLabel,
  FieldLegend,
  FieldSet,
} from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import { RegisterFormData, registerFormSchema } from "@/lib/schemas/registerUserSchema"
import { zodResolver } from "@hookform/resolvers/zod"
import { useForm, SubmitHandler } from "react-hook-form"
import { Spinner } from "@/components/ui/spinner"
import { type APIResponse, showAPIResponseToast } from "@/components/ui/showToast"
import axios from "axios"

function isAPIResponse(payload: unknown): payload is APIResponse {
    if (!payload || typeof payload !== "object") {
        return false
    }
    const candidate = payload as Record<string, unknown>
    return (
        (candidate.type === "success" ||
            candidate.type === "info" ||
            candidate.type === "error") &&
        typeof candidate.title === "string" &&
        typeof candidate.message === "string"
    )
}

export default function RegistrationForm() {
    const {register, handleSubmit, reset, formState : {errors, isSubmitting}, setError} = useForm<RegisterFormData>({
        resolver : zodResolver(registerFormSchema), 
        defaultValues : {
            email : '',
            confirmEmail : '',
            password : '',
            confirmPassword : '',
        },
        mode : 'onTouched'
    })

    //Submit the form
    const onSubmit : SubmitHandler<RegisterFormData> = async (data) => {
        try {
            const response = await axios.post(
                '/api/auth/register',
                data
            )
            const toastPayload: APIResponse = isAPIResponse(response.data)
                ? response.data
                : {
                    type: "success",
                    title: "Registration Successful",
                    message: "Account created. Please check your email to verify your address.",
                }
            showAPIResponseToast(toastPayload)
            reset()
        } catch(error) {
            const responseData = axios.isAxiosError(error) ? error.response?.data : undefined
            const toastPayload: APIResponse = isAPIResponse(responseData)
                ? responseData
                : {
                    type: "error",
                    title: "Registration Failed",
                    message:
                        axios.isAxiosError(error) && typeof error.response?.data?.message === "string"
                            ? error.response.data.message
                            : "Unable to complete registration. Please try again.",
                }
            showAPIResponseToast(toastPayload)
            setError("root", { message: toastPayload.message })
        }
    }
    return (
        <div className="w-full">
            <form className="w-full" onSubmit={handleSubmit(onSubmit)}>
                <FieldGroup>
                    <FieldSet>
                        <FieldLegend>Create your account</FieldLegend>
                        <FieldDescription>All fields are required</FieldDescription>
                        <FieldGroup>
                            <Field>
                                <FieldLabel htmlFor="email">Email address</FieldLabel>
                                <Input
                                id="email"
                                type="email"
                                placeholder="john@example.com"
                                required
                                {...register('email')}
                                />
                                {errors.email && (
                                    <FieldError>{errors.email.message}</FieldError>
                                )}
                            </Field>
                            <Field>
                                <FieldLabel htmlFor="confirmemail">Confirm email address</FieldLabel>
                                <Input
                                id="confirmemail"
                                type="email"
                                placeholder="john@example.com"
                                required
                                {...register('confirmEmail')}
                                />
                                {errors.confirmEmail && (
                                    <FieldError>{errors.confirmEmail.message}</FieldError>
                                )}
                            </Field>
                            <Field>
                                <FieldLabel htmlFor="password">Password</FieldLabel>
                                <Input
                                id="password"
                                placeholder="*************"
                                type="password"
                                required
                                {...register('password')}
                                />
                                <FieldDescription>Min 8 Characters</FieldDescription>
                                {errors.password && (
                                    <FieldError>{errors.password.message}</FieldError>
                                )}
                            </Field>
                            <Field>
                                <FieldLabel htmlFor="passwordconfirm">Confirm password</FieldLabel>
                                <Input
                                id="passwordconfirm"
                                placeholder="*************"
                                type="password"
                                required
                                {...register('confirmPassword')}
                                />
                                {errors.confirmPassword && (
                                    <FieldError>{errors.confirmPassword.message}</FieldError>
                                )}
                            </Field>
                        </FieldGroup>
                    </FieldSet>
                    {errors.root && (
                        <FieldError>{errors.root.message}</FieldError>
                    )}
                    <Field orientation="horizontal" className="pt-1">
                        <Button type="submit" className="flex-1" disabled={isSubmitting}>
                            {isSubmitting && <Spinner/>}
                            {isSubmitting ? 'Registering...' : "Register"}
                        </Button>
                        <Button variant="outline" type="reset" className="flex-1" onClick={() => reset()}>
                            Clear
                        </Button>
                    </Field>
                </FieldGroup>
            </form>
        </div>
    )
}
