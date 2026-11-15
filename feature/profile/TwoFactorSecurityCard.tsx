// Interactive TOTP Two-Factor Authentication (2FA) management card and dialogs.
// Wired to POST /api/auth/2fa/setup, POST /api/auth/2fa/enable, and POST /api/auth/2fa/disable.
"use client"

import { useState } from "react"
import axios from "axios"
import { useRouter } from "next/navigation"
import { useForm, type SubmitHandler } from "react-hook-form"
import { zodResolver } from "@hookform/resolvers/zod"
import { Check, Copy, KeyRound, ShieldAlert, ShieldCheck } from "lucide-react"
import { Button } from "@/components/ui/button"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { Field, FieldDescription, FieldError, FieldGroup, FieldLabel } from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import { showAPIResponseToast } from "@/components/ui/showToast"
import { Spinner } from "@/components/ui/spinner"
import {
  twoFactorCodeSchema,
  type TwoFactorCodeFormData,
  type TwoFactorSetupPayload,
} from "@/lib/schemas/twoFactorSchema"

interface TwoFactorSecurityCardProps {
  initialEnabled: boolean
  onStatusChange?: (enabled: boolean) => void
}

export function TwoFactorSecurityCard({
  initialEnabled,
  onStatusChange,
}: TwoFactorSecurityCardProps) {
  const router = useRouter()
  const [isEnabled, setIsEnabled] = useState(initialEnabled)
  const [isStartingSetup, setIsStartingSetup] = useState(false)
  const [setupDialogOpen, setSetupDialogOpen] = useState(false)
  const [disableDialogOpen, setDisableDialogOpen] = useState(false)
  const [setupData, setSetupData] = useState<TwoFactorSetupPayload | null>(null)
  const [copiedSecret, setCopiedSecret] = useState(false)

  const enableForm = useForm<TwoFactorCodeFormData>({
    resolver: zodResolver(twoFactorCodeSchema),
    defaultValues: { code: "" },
    mode: "onSubmit",
  })

  const disableForm = useForm<TwoFactorCodeFormData>({
    resolver: zodResolver(twoFactorCodeSchema),
    defaultValues: { code: "" },
    mode: "onSubmit",
  })

  const handleStartSetup = async () => {
    setIsStartingSetup(true)
    enableForm.reset({ code: "" })
    setCopiedSecret(false)

    try {
      const response = await axios.post<TwoFactorSetupPayload>(
        "/api/auth/2fa/setup",
        {},
        { withCredentials: true }
      )
      setSetupData(response.data)
      setSetupDialogOpen(true)
    } catch (error) {
      if (axios.isAxiosError(error) && error.response?.status === 401) {
        router.replace("/")
        router.refresh()
        return
      }
      const message =
        axios.isAxiosError(error) && typeof error.response?.data?.error === "string"
          ? error.response.data.error
          : "Unable to start 2FA setup right now."
      showAPIResponseToast({
        type: "error",
        title: "2FA Setup Failed",
        message,
      })
    } finally {
      setIsStartingSetup(false)
    }
  }

  const handleCopySecret = async () => {
    if (!setupData?.secret) {
      return
    }
    try {
      await navigator.clipboard.writeText(setupData.secret)
      setCopiedSecret(true)
      setTimeout(() => setCopiedSecret(false), 2000)
    } catch {
      showAPIResponseToast({
        type: "error",
        title: "Copy Failed",
        message: "Copy the secret key manually from the field below.",
      })
    }
  }

  const handleEnableSubmit: SubmitHandler<TwoFactorCodeFormData> = async (data) => {
    try {
      await axios.post(
        "/api/auth/2fa/enable",
        { code: data.code },
        { withCredentials: true }
      )
      setIsEnabled(true)
      onStatusChange?.(true)
      setSetupDialogOpen(false)
      setSetupData(null)
      enableForm.reset({ code: "" })
      showAPIResponseToast({
        type: "success",
        title: "Two-Factor Authentication Enabled",
        message: "Your account is now protected with TOTP 2FA.",
      })
      router.refresh()
    } catch (error) {
      if (axios.isAxiosError(error) && error.response?.status === 401) {
        router.replace("/")
        router.refresh()
        return
      }
      const message =
        axios.isAxiosError(error) && typeof error.response?.data?.error === "string"
          ? error.response.data.error
          : "Invalid authenticator code. Please try again."
      enableForm.setError("root", { message })
    }
  }

  const handleDisableSubmit: SubmitHandler<TwoFactorCodeFormData> = async (data) => {
    try {
      await axios.post(
        "/api/auth/2fa/disable",
        { code: data.code },
        { withCredentials: true }
      )
      setIsEnabled(false)
      onStatusChange?.(false)
      setDisableDialogOpen(false)
      disableForm.reset({ code: "" })
      showAPIResponseToast({
        type: "success",
        title: "Two-Factor Authentication Disabled",
        message: "TOTP 2FA has been turned off for your account.",
      })
      router.refresh()
    } catch (error) {
      if (axios.isAxiosError(error) && error.response?.status === 401) {
        router.replace("/")
        router.refresh()
        return
      }
      const message =
        axios.isAxiosError(error) && typeof error.response?.data?.error === "string"
          ? error.response.data.error
          : "Invalid authenticator code. Unable to disable 2FA."
      disableForm.setError("root", { message })
    }
  }

  return (
    <section className="w-full rounded-xl border border-border bg-card p-6 shadow-xs">
      <div className="flex flex-col justify-between gap-4 sm:flex-row sm:items-center">
        <div className="space-y-1">
          <div className="flex items-center gap-2">
            {isEnabled ? (
              <ShieldCheck className="size-5 text-emerald-600 dark:text-emerald-400" />
            ) : (
              <ShieldAlert className="size-5 text-amber-600 dark:text-amber-400" />
            )}
            <h2 className="text-lg font-semibold tracking-tight">
              Two-Factor Authentication (TOTP)
            </h2>
            <span
              className={
                isEnabled
                  ? "rounded-full bg-emerald-500/10 px-2.5 py-0.5 text-xs font-medium text-emerald-700 dark:text-emerald-300"
                  : "rounded-full bg-amber-500/10 px-2.5 py-0.5 text-xs font-medium text-amber-700 dark:text-amber-300"
              }
            >
              {isEnabled ? "Enabled" : "Disabled"}
            </span>
          </div>
          <p className="text-sm text-muted-foreground">
            {isEnabled
              ? "Your account requires a 6-digit authenticator code each time you sign in."
              : "Protect your health data with an authenticator app such as Google Authenticator, 1Password, or Authy."}
          </p>
        </div>

        <div className="shrink-0">
          {isEnabled ? (
            <Button
              type="button"
              variant="destructive"
              onClick={() => {
                disableForm.reset({ code: "" })
                setDisableDialogOpen(true)
              }}
            >
              Disable 2FA
            </Button>
          ) : (
            <Button
              type="button"
              disabled={isStartingSetup}
              onClick={handleStartSetup}
            >
              {isStartingSetup && <Spinner />}
              <KeyRound className="size-4" />
              {isStartingSetup ? "Preparing Setup..." : "Enable 2FA"}
            </Button>
          )}
        </div>
      </div>

      {/* Setup & Enable 2FA Dialog */}
      <Dialog
        open={setupDialogOpen}
        onOpenChange={(open) => {
          setSetupDialogOpen(open)
          if (!open) {
            setSetupData(null)
            enableForm.reset({ code: "" })
          }
        }}
      >
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle>Set up Authenticator App</DialogTitle>
            <DialogDescription>
              Scan the QR code below with your authenticator app, or enter the manual secret key, then confirm with a 6-digit code.
            </DialogDescription>
          </DialogHeader>

          {setupData && (
            <div className="space-y-4">
              <div className="flex justify-center rounded-lg border border-border bg-white p-4">
                {/* eslint-disable-next-line @next/next/no-img-element */}
                <img
                  src={setupData.qrCode}
                  alt="TOTP Two-Factor Authentication QR Code"
                  className="size-44 object-contain"
                />
              </div>

              <div className="space-y-1.5">
                <p className="text-xs font-medium text-muted-foreground">
                  Manual Setup Key
                </p>
                <div className="flex items-center gap-2">
                  <Input
                    readOnly
                    value={setupData.secret}
                    aria-label="TOTP Manual Secret Key"
                    className="font-mono text-xs"
                  />
                  <Button
                    type="button"
                    variant="outline"
                    size="icon"
                    onClick={handleCopySecret}
                    aria-label="Copy manual secret key"
                  >
                    {copiedSecret ? (
                      <Check className="size-4 text-emerald-600" />
                    ) : (
                      <Copy className="size-4" />
                    )}
                  </Button>
                </div>
                {setupData.uri && (
                  <p className="text-xs text-muted-foreground">
                    Or open directly:{" "}
                    <a
                      href={setupData.uri}
                      className="underline underline-offset-2 hover:text-foreground"
                    >
                      Open in Authenticator App
                    </a>
                  </p>
                )}
              </div>

              <form
                onSubmit={enableForm.handleSubmit(handleEnableSubmit)}
                className="space-y-4"
              >
                <FieldGroup>
                  <Field>
                    <FieldLabel htmlFor="enable-2fa-code">
                      6-Digit Verification Code
                    </FieldLabel>
                    <Input
                      id="enable-2fa-code"
                      inputMode="numeric"
                      autoComplete="one-time-code"
                      maxLength={6}
                      placeholder="123456"
                      {...enableForm.register("code")}
                    />
                    <FieldDescription>
                      Enter the 6-digit code currently shown in your authenticator app.
                    </FieldDescription>
                    {enableForm.formState.errors.code && (
                      <FieldError>
                        {enableForm.formState.errors.code.message}
                      </FieldError>
                    )}
                  </Field>
                </FieldGroup>

                {enableForm.formState.errors.root && (
                  <FieldError>
                    {enableForm.formState.errors.root.message}
                  </FieldError>
                )}

                <DialogFooter>
                  <Button
                    type="button"
                    variant="outline"
                    disabled={enableForm.formState.isSubmitting}
                    onClick={() => setSetupDialogOpen(false)}
                  >
                    Cancel
                  </Button>
                  <Button
                    type="submit"
                    disabled={enableForm.formState.isSubmitting}
                  >
                    {enableForm.formState.isSubmitting && <Spinner />}
                    {enableForm.formState.isSubmitting
                      ? "Verifying..."
                      : "Verify & Enable"}
                  </Button>
                </DialogFooter>
              </form>
            </div>
          )}
        </DialogContent>
      </Dialog>

      {/* Disable 2FA Dialog */}
      <Dialog
        open={disableDialogOpen}
        onOpenChange={(open) => {
          setDisableDialogOpen(open)
          if (!open) {
            disableForm.reset({ code: "" })
          }
        }}
      >
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle>Disable Two-Factor Authentication</DialogTitle>
            <DialogDescription>
              Enter a valid 6-digit code from your authenticator app to confirm disabling 2FA on your account.
            </DialogDescription>
          </DialogHeader>

          <form
            onSubmit={disableForm.handleSubmit(handleDisableSubmit)}
            className="space-y-4"
          >
            <FieldGroup>
              <Field>
                <FieldLabel htmlFor="disable-2fa-code">
                  6-Digit Verification Code
                </FieldLabel>
                <Input
                  id="disable-2fa-code"
                  inputMode="numeric"
                  autoComplete="one-time-code"
                  maxLength={6}
                  placeholder="123456"
                  {...disableForm.register("code")}
                />
                {disableForm.formState.errors.code && (
                  <FieldError>
                    {disableForm.formState.errors.code.message}
                  </FieldError>
                )}
              </Field>
            </FieldGroup>

            {disableForm.formState.errors.root && (
              <FieldError>
                {disableForm.formState.errors.root.message}
              </FieldError>
            )}

            <DialogFooter>
              <Button
                type="button"
                variant="outline"
                disabled={disableForm.formState.isSubmitting}
                onClick={() => setDisableDialogOpen(false)}
              >
                Cancel
              </Button>
              <Button
                type="submit"
                variant="destructive"
                disabled={disableForm.formState.isSubmitting}
              >
                {disableForm.formState.isSubmitting && <Spinner />}
                {disableForm.formState.isSubmitting
                  ? "Disabling..."
                  : "Confirm Disable"}
              </Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>
    </section>
  )
}
