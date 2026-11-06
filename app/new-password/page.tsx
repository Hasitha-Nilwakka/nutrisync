"use client"
import { Button } from "@/components/ui/button"
import { FieldGroup, FieldLegend, FieldSet, FieldDescription, Field, FieldLabel, FieldSeparator, FieldError } from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import { showAPIResponseToast, showRedirectToast } from "@/components/ui/showToast"
import { PasswordChangeFormData, passwordChangeSchema } from "@/lib/schemas/passwordChangeSchema"
import { zodResolver } from "@hookform/resolvers/zod"
import axios from "axios"
import { useParams, useRouter } from "next/navigation"
import { useForm, SubmitHandler} from "react-hook-form"

export default function ResetPasswordPage() {
    const { token } = useParams()
    const router = useRouter()

    const {register,handleSubmit, formState : {errors, isSubmitting}} = useForm<PasswordChangeFormData>({
        resolver : zodResolver(passwordChangeSchema),
        defaultValues : {
            password : '',
            confirmPassword : ''
        },
        mode : 'all'
    })

    /*
    if (!token) {
        showRedirectToast({
            title : 'Unauthorized action',
            message : 'This action seems unauthorized',
            redirectTo : 'login page',
            type : 'error'
        })
        router.replace('/')
        router.refresh()
    }*/

   const onSubmit : SubmitHandler<PasswordChangeFormData> = async (data) => {
        try {
            const response = await axios.post(
                'api/auth/passwordreset',
                {
                    token : token,
                    newPassword : data.password
                }
            )
            showRedirectToast({
                type : 'success',
                title : response?.data?.title,
                message : response.data.message,
                redirectTo : 'login page'
            })
            router.replace('/')
            router.refresh()
        } catch(err) {
            if(axios.isAxiosError(err) && err.response) {
                console.log(err)
                showAPIResponseToast(err.response.data)
            }
        }
   }

    return (
        <div className="w-full max-w-md">
            <form onSubmit={handleSubmit(onSubmit)}>
                <FieldGroup className="border p-2">
                    <FieldSet>
                        <FieldLegend>Create a new password</FieldLegend>
                        <FieldDescription>You are chaning your password</FieldDescription>
                        <FieldGroup>
                            <Field>
                                <FieldLabel>Password</FieldLabel>
                                <Input
                                    {...register('password')}
                                    type='password'
                                    placeholder="********"
                                />
                                {errors.password && <FieldError>{errors.password.message}</FieldError>}
                            </Field>
                            <Field>
                                <FieldLabel>Confirm password</FieldLabel>
                                <Input
                                    {...register('confirmPassword')}
                                    type='password'
                                    placeholder="********"
                                />
                                {errors.confirmPassword && <FieldError>{errors.confirmPassword.message}</FieldError>}
                            </Field>
                        </FieldGroup>
                        <FieldSeparator/>
                        <Field orientation={'horizontal'}>
                            <Button type='submit' disabled={isSubmitting}>Submit</Button>
                            <Button variant={'secondary'} type='reset' disabled={isSubmitting}>Cancel</Button>
                        </Field>
                    </FieldSet>
                </FieldGroup>
            </form>
        </div>
    )
}