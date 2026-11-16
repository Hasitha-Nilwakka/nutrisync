// SendPasswordResetVerification renders the "Forgot password?" trigger and modal dialog
// to request a password reset verification link via POST /api/auth/passwordreset.
"use client"

import { useState } from "react"
import { zodResolver } from "@hookform/resolvers/zod"
import axios from "axios"
import { useRouter } from "next/navigation"
import { type SubmitHandler, useForm } from "react-hook-form"
import z from "zod"
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
import { type APIResponse, showAPIResponseToast } from "@/components/ui/showToast"
import { Spinner } from "@/components/ui/spinner"

const emailSchema = z.object({
  emailReset: z
    .string()
    .trim()
    .min(1, "Email address is required")
    .email("Invalid email address format"),
})

type EmailSchema = z.infer<typeof emailSchema>

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

function extractErrorToast(error: unknown): APIResponse {
  if (axios.isAxiosError(error)) {
    const responseData = error.response?.data
    if (isAPIResponse(responseData)) {
      return responseData
    }
    if (responseData && typeof responseData === "object") {
      const record = responseData as Record<string, unknown>
      const fallbackMessage =
        typeof record.message === "string"
          ? record.message
          : typeof record.error === "string"
            ? record.error
            : null
      if (fallbackMessage) {
        return {
          type: "error",
          title: "Password Reset Failed",
          message: fallbackMessage,
        }
      }
    }
    return {
      type: "error",
      title: "Unable to Send Reset Link",
      message: "Please check your connection and try again.",
    }
  }

  return {
    type: "error",
    title: "Unable to Send Reset Link",
    message: "An unexpected error occurred. Please try again.",
  }
}

export function SendPasswordResetVerification() {
  const router = useRouter()
  const [open, setOpen] = useState(false)
  const {
    register,
    handleSubmit,
    reset,
    formState: { errors, isSubmitting },
    clearErrors,
  } = useForm<EmailSchema>({
    defaultValues: {
      emailReset: "",
    },
    resolver: zodResolver(emailSchema),
    mode: "onTouched",
  })

  const onSubmit: SubmitHandler<EmailSchema> = async (data) => {
    try {
      const response = await axios.post("/api/auth/passwordreset", {
        emailReset: data.emailReset,
      })
      const toastPayload: APIResponse = isAPIResponse(response.data)
        ? response.data
        : {
            type: "success",
            title: "Password Reset Link Sent",
            message:
              "If an account exists for that email, a password reset link has been sent.",
          }

      showAPIResponseToast(toastPayload)
      if (toastPayload.type !== "error") {
        reset()
        setOpen(false)
        router.replace("/")
        router.refresh()
      }
    } catch (err) {
      showAPIResponseToast(extractErrorToast(err))
    }
  }

  return (
    <Dialog
      open={open}
      disablePointerDismissal
      onOpenChange={(nextOpen) => {
        setOpen(nextOpen)
        if (!nextOpen) {
          reset()
          clearErrors("emailReset")
        }
      }}
    >
      <DialogTrigger
        render={
          <Button
            variant="link"
            className="h-auto px-0 text-xs font-medium text-muted-foreground underline underline-offset-4 hover:text-primary"
          >
            Forgot password?
          </Button>
        }
      />
      <DialogContent className="overflow-hidden p-0 sm:max-w-md">
        <div
          aria-hidden="true"
          className="h-1.5 w-full bg-linear-to-r from-[#0B74B8] via-[#18A999] to-[#43B047]"
        />
        <form onSubmit={handleSubmit(onSubmit)} className="space-y-5 p-6 pt-5">
          <DialogHeader className="space-y-1.5 text-left">
            <DialogTitle className="text-lg font-semibold">
              Reset your password
            </DialogTitle>
            <DialogDescription className="text-sm leading-relaxed">
              Enter the email address associated with your account and we&apos;ll
              send you a password reset link.
            </DialogDescription>
          </DialogHeader>

          <FieldGroup>
            <Field className="gap-2">
              <Label htmlFor="email-reset">Email address</Label>
              <Input
                id="email-reset"
                type="email"
                placeholder="john@example.com"
                autoComplete="email"
                {...register("emailReset")}
              />
              {errors.emailReset && (
                <FieldError>{errors.emailReset.message}</FieldError>
              )}
            </Field>
          </FieldGroup>

          <DialogFooter className="gap-2 pt-1 sm:justify-end">
            <DialogClose
              render={
                <Button type="button" variant="outline" disabled={isSubmitting}>
                  Cancel
                </Button>
              }
            />
            <Button type="submit" disabled={isSubmitting}>
              {isSubmitting && <Spinner />}
              {isSubmitting ? "Sending..." : "Send Reset Link"}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}
