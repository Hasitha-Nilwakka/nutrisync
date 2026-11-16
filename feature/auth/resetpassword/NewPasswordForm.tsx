"use client"

import Link from "next/link"
import { useRouter, useSearchParams } from "next/navigation"
import axios from "axios"
import { useForm, SubmitHandler } from "react-hook-form"
import { zodResolver } from "@hookform/resolvers/zod"
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
import {
  type APIResponse,
  showAPIResponseToast,
  showRedirectToast,
} from "@/components/ui/showToast"
import {
  type PasswordChangeFormData,
  passwordChangeSchema,
} from "@/lib/schemas/passwordChangeSchema"

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

function extractErrorToast(err: unknown): APIResponse {
  if (axios.isAxiosError(err)) {
    const responseData = err.response?.data
    if (isAPIResponse(responseData)) {
      return responseData
    }
    if (
      responseData &&
      typeof responseData === "object" &&
      typeof (responseData as { error?: unknown }).error === "string"
    ) {
      return {
        type: "error",
        title: "Password Reset Failed",
        message: (responseData as { error: string }).error,
      }
    }
  }

  return {
    type: "error",
    title: "Password Reset Failed",
    message: "Unable to reset your password right now. Please try again.",
  }
}

export function NewPasswordForm() {
  const router = useRouter()
  const searchParams = useSearchParams()
  const token = searchParams.get("token")?.trim() ?? ""

  const {
    register,
    handleSubmit,
    reset,
    formState: { errors, isSubmitting },
  } = useForm<PasswordChangeFormData>({
    resolver: zodResolver(passwordChangeSchema),
    defaultValues: {
      password: "",
      confirmPassword: "",
    },
    mode: "onTouched",
  })

  if (!token) {
    return (
      <div className="w-full max-w-md overflow-hidden rounded-2xl border border-border/80 bg-card shadow-sm">
        <div
          aria-hidden="true"
          className="h-1.5 w-full bg-linear-to-r from-[#0B74B8] via-[#18A999] to-[#43B047]"
        />
        <div className="space-y-4 p-6">
          <h1 className="text-lg font-semibold text-foreground">
            Invalid or Missing Reset Link
          </h1>
          <p className="text-sm leading-relaxed text-muted-foreground">
            This password reset link is missing a valid verification token.
            Please return to the login page and request a new reset link.
          </p>
          <div className="pt-2">
            <Link
              href="/"
              className="inline-flex h-10 w-full items-center justify-center rounded-lg bg-primary px-4 text-sm font-medium text-primary-foreground transition-colors hover:bg-primary/90"
            >
              Back to Log In
            </Link>
          </div>
        </div>
      </div>
    )
  }

  const onSubmit: SubmitHandler<PasswordChangeFormData> = async (data) => {
    try {
      const response = await axios.post("/api/auth/passwordreset/confirm", {
        token,
        newPassword: data.password,
      })

      const payload: APIResponse = isAPIResponse(response.data)
        ? response.data
        : {
            type: "success",
            title: "Password Updated",
            message:
              "Your password has been reset. You can now log in with your new password.",
          }

      showRedirectToast({
        type: payload.type,
        title: payload.title,
        message: payload.message,
        redirectTo: "login page",
      })
      reset()
      router.replace("/")
      router.refresh()
    } catch (err) {
      showAPIResponseToast(extractErrorToast(err))
    }
  }

  return (
    <div className="w-full max-w-md overflow-hidden rounded-2xl border border-border/80 bg-card shadow-sm">
      <div
        aria-hidden="true"
        className="h-1.5 w-full bg-linear-to-r from-[#0B74B8] via-[#18A999] to-[#43B047]"
      />
      <form onSubmit={handleSubmit(onSubmit)} className="p-6">
        <FieldGroup className="gap-5">
          <FieldSet>
            <FieldLegend>Create a new password</FieldLegend>
            <FieldDescription>
              Choose a strong password of at least 8 characters for your
              NutriSync account.
            </FieldDescription>
            <FieldGroup className="mt-2 gap-4">
              <Field>
                <FieldLabel htmlFor="new-password">New password</FieldLabel>
                <Input
                  id="new-password"
                  {...register("password")}
                  type="password"
                  autoComplete="new-password"
                  placeholder="Minimum 8 characters"
                />
                {errors.password && (
                  <FieldError>{errors.password.message}</FieldError>
                )}
              </Field>
              <Field>
                <FieldLabel htmlFor="confirm-new-password">
                  Confirm new password
                </FieldLabel>
                <Input
                  id="confirm-new-password"
                  {...register("confirmPassword")}
                  type="password"
                  autoComplete="new-password"
                  placeholder="Re-enter your new password"
                />
                {errors.confirmPassword && (
                  <FieldError>{errors.confirmPassword.message}</FieldError>
                )}
              </Field>
            </FieldGroup>
            <FieldSeparator />
            <Field orientation="horizontal" className="gap-3 pt-1">
              <Button
                type="submit"
                disabled={isSubmitting}
                className="h-10 flex-1 rounded-lg"
              >
                {isSubmitting ? "Updating..." : "Update Password"}
              </Button>
              <Button
                variant="outline"
                type="button"
                disabled={isSubmitting}
                className="h-10 flex-1 rounded-lg"
                onClick={() => router.replace("/")}
              >
                Cancel
              </Button>
            </Field>
          </FieldSet>
        </FieldGroup>
      </form>
    </div>
  )
}
