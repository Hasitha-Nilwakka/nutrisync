"use client"
import { Button } from "@/components/ui/button"
import { Checkbox } from "@/components/ui/checkbox"
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
import { RegisterFormData, registerFormSchema } from "@/lib/schemas/registerUserSchema"
import { zodResolver } from "@hookform/resolvers/zod"
import { useForm, SubmitHandler, Controller } from "react-hook-form"
import { Spinner } from "@/components/ui/spinner"
import axios from "axios"

export default function RegistrationForm() {
    const {register, handleSubmit, control, reset, formState : {errors, isSubmitting}, setError} = useForm<RegisterFormData>({
        resolver : zodResolver(registerFormSchema), 
        defaultValues : {
            email : '',
            confirmEmail : '',
            password : '',
            confirmPassword : '',
            concented : false
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
                    <FieldSeparator />
                    <FieldSet>
                        <FieldGroup>
                            <Field orientation="horizontal">
                                <Controller 
                                    control={control}
                                    name="concented"
                                    render={({field : {onChange, value}}) => (
                                        <Checkbox
                                            id="concent"
                                            onCheckedChange={onChange}
                                            checked={value}
                                        />
                                    )}
                                />
                                <FieldLabel htmlFor="concent" className="font-normal">
                                I consent to the processing of my personal and health-related data.
                                </FieldLabel>
                            </Field>
                            {errors.concented && (
                                <FieldError>{errors.concented.message}</FieldError>
                            )}
                            <FieldDescription>
                                I understand that NutriSync collects and processes my profile information, 
                                health metrics, fitness goals, and lifestyle preferences to calculate wellness indicators, 
                                track my progress, and generate personalized health insights using AI. 
                                I have read the [Privacy Policy] and understand how my data is used, stored, and protected. 
                                I understand that I can withdraw my consent at any time.
                            </FieldDescription>
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