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
import axios from "axios"

export default function RegistrationForm() {
    const {register, handleSubmit, reset, formState : {errors, isSubmitting}, setError} = useForm<RegisterFormData>({
        resolver : zodResolver(registerFormSchema), 
        defaultValues : {
            email : '',
            confirmEmail : '',
            password : '',
            confirmPassword : '',
        },
        mode : 'all'
    })

    //Submit the form
    const onSubmit : SubmitHandler<RegisterFormData> = async (data) => {
        try {
            await axios.post(
                '/api/auth/register',
                data
            )
            alert('user registered')
        } catch(error) {
            const message = axios.isAxiosError(error) ? error.response?.data?.message : "Registration failed"
            setError("root", {message : message})
        }
    }
    return (
        <div>
            <form className="w-full max-w-md border" onSubmit={handleSubmit(onSubmit)}>
                <FieldGroup>
                    <FieldSet>
                        <FieldLegend>Register as a new user</FieldLegend>
                        <FieldDescription>All fields are mandatory</FieldDescription>
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
                    <Field orientation="horizontal">
                        <Button type="submit" disabled={isSubmitting}>
                            {isSubmitting && <Spinner/>}
                            {isSubmitting ? 'Registering' : "Register"}
                        </Button>
                        <Button variant="outline" type="reset" onClick={() => reset()}>
                            Clear
                        </Button>
                    </Field>
                </FieldGroup>
            </form>
        </div>
    )
}
