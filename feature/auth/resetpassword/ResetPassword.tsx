//send the reset verification link to user
"use client"
import { Button } from "@/components/ui/button"
import {
  Dialog,
  DialogClose,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog"
import { Field, FieldError, FieldGroup } from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { zodResolver } from "@hookform/resolvers/zod"
import { SubmitHandler, useForm } from "react-hook-form"
import z from "zod"
import { useRouter } from "next/navigation"
import axios from "axios"
import { showToast } from "@/components/ui/showToast"

const emailSchema = z.object({
    emailReset : z.string().trim().min(1, 'Email address is required').email('Invalid email address format')
})

type EmailSchema = z.infer<typeof emailSchema>
export function SendPasswordResetVerification() {
    const router = useRouter()
    const {register,handleSubmit, formState : {errors, isSubmitting}, clearErrors} = useForm<EmailSchema>({
        defaultValues : {
            emailReset : ''
        },
        resolver : zodResolver(emailSchema),
        mode : 'onSubmit'
    })

    const onSubmit : SubmitHandler<EmailSchema> = async(data) => {
        try {
            const response = await axios.post(
                '/api/auth/passwordreset'
                ,{emailReset : data.emailReset}
            )
            showToast(response.data)
            router.replace('/')
            router.refresh()
        } catch(err) {
            if(axios.isAxiosError(err) && err.response) {
                console.log(err)
                showToast(err.response.data)
            }
        }

    }
  return (
    <Dialog disablePointerDismissal onOpenChange={() => clearErrors('emailReset')}>
        <DialogTrigger render={<Button variant='link'>forgot password?</Button>} />
        <DialogContent className="sm:max-w-sm">
        <form onSubmit={handleSubmit(onSubmit)}>
            <DialogHeader>
                <DialogTitle>Password Rest</DialogTitle>
                <DialogDescription>
                Enter your email address to receive a password reset verification link.
                </DialogDescription>
            </DialogHeader>
            <FieldGroup>
                <Field>
                <Label htmlFor="email-reset">Email</Label>
                <Input 
                    id="email-reset" 
                    placeholder="john@example.com" 
                    {...register('emailReset')}
                    />
                    {errors.emailReset && <FieldError>{errors.emailReset?.message}</FieldError>}
                </Field>
            </FieldGroup>
            <DialogFooter>
                <DialogClose render={<Button variant="outline">Cancel</Button>} />
                <Button type="submit" disabled={isSubmitting}>Send verification</Button>
            </DialogFooter>
        </form>
        </DialogContent>
    </Dialog>
  )
}
