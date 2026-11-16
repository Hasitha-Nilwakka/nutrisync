"use client"

import { useState } from "react"
import { useRouter } from "next/navigation"
import axios from "axios"
import { AlertTriangle, ShieldAlert, Trash2 } from "lucide-react"
import { Button } from "@/components/ui/button"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { showAPIResponseToast, showRedirectToast } from "@/components/ui/showToast"

interface PrivacyAndAccountDeletionCardProps {
  userEmail: string
}

export function PrivacyAndAccountDeletionCard({
  userEmail,
}: PrivacyAndAccountDeletionCardProps) {
  const router = useRouter()
  const [dialogOpen, setDialogOpen] = useState(false)
  const [triggerReason, setTriggerReason] = useState<"withdraw" | "delete">("delete")
  const [confirmationText, setConfirmationText] = useState("")
  const [isDeleting, setIsDeleting] = useState(false)

  const openConfirmationModal = (reason: "withdraw" | "delete") => {
    setTriggerReason(reason)
    setConfirmationText("")
    setDialogOpen(true)
  }

  const handleConfirmDeleteAccount = async () => {
    if (confirmationText.trim() !== "DELETE") {
      return
    }

    setIsDeleting(true)
    try {
      const response = await axios.delete("/api/auth/account", {
        withCredentials: true,
      })

      showRedirectToast({
        type: "success",
        title:
          typeof response.data?.title === "string"
            ? response.data.title
            : "Account Deleted",
        message:
          typeof response.data?.message === "string"
            ? response.data.message
            : "Your account and all health data have been permanently deleted.",
        redirectTo: "home page",
      })
      setDialogOpen(false)
      router.replace("/")
      router.refresh()
    } catch (error) {
      const message =
        axios.isAxiosError(error) && typeof error.response?.data?.message === "string"
          ? error.response.data.message
          : "Unable to delete your account right now. Please try again."
      showAPIResponseToast({
        type: "error",
        title: "Account Deletion Failed",
        message,
      })
    } finally {
      setIsDeleting(false)
    }
  }

  return (
    <section className="rounded-2xl border border-destructive/30 bg-card p-6 shadow-sm">
      <div className="flex flex-col gap-5">
        <div className="flex flex-wrap items-start justify-between gap-4 border-b border-border/60 pb-4">
          <div className="space-y-1">
            <div className="flex items-center gap-2">
              <ShieldAlert className="size-5 text-destructive" />
              <h2 className="text-lg font-semibold tracking-tight text-foreground">
                GDPR Consent &amp; Account Deletion
              </h2>
            </div>
            <p className="text-sm text-muted-foreground">
              Manage your mandatory GDPR data-processing consent or permanently erase your NutriSync account.
            </p>
          </div>
          <span className="inline-flex items-center rounded-full border border-emerald-500/30 bg-emerald-500/10 px-2.5 py-0.5 text-xs font-medium text-emerald-700 dark:text-emerald-300">
            GDPR Consent Granted
          </span>
        </div>

        <div className="grid gap-4 sm:grid-cols-2">
          <div className="flex flex-col justify-between gap-3 rounded-xl border border-border/80 bg-muted/15 p-4">
            <div className="space-y-1">
              <h3 className="text-sm font-semibold text-foreground">
                Withdraw Mandatory GDPR Consent
              </h3>
              <p className="text-xs leading-relaxed text-muted-foreground">
                NutriSync requires GDPR consent to store and process biometric data. Withdrawing consent will permanently delete your profile and account.
              </p>
            </div>
            <div>
              <Button
                type="button"
                variant="outline"
                size="sm"
                className="border-destructive/40 text-destructive hover:bg-destructive/10 hover:text-destructive"
                onClick={() => openConfirmationModal("withdraw")}
              >
                Withdraw Consent
              </Button>
            </div>
          </div>

          <div className="flex flex-col justify-between gap-3 rounded-xl border border-destructive/30 bg-destructive/5 p-4">
            <div className="space-y-1">
              <h3 className="text-sm font-semibold text-foreground">
                Delete Profile &amp; Account
              </h3>
              <p className="text-xs leading-relaxed text-muted-foreground">
                Permanently remove <span className="font-medium text-foreground">{userEmail}</span>, your health profile, weight &amp; activity logs, AI insights, and active sessions.
              </p>
            </div>
            <div>
              <Button
                type="button"
                variant="destructive"
                size="sm"
                onClick={() => openConfirmationModal("delete")}
              >
                <Trash2 className="size-4" />
                Delete Account &amp; Data
              </Button>
            </div>
          </div>
        </div>
      </div>

      <Dialog
        open={dialogOpen}
        onOpenChange={(nextOpen) => {
          if (!isDeleting) {
            setDialogOpen(nextOpen)
            if (!nextOpen) {
              setConfirmationText("")
            }
          }
        }}
      >
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle className="flex items-center gap-2 text-destructive">
              <AlertTriangle className="size-5" />
              {triggerReason === "withdraw"
                ? "Withdraw GDPR Consent & Delete Account"
                : "Permanently Delete Account & Profile"}
            </DialogTitle>
            <DialogDescription className="text-left leading-relaxed">
              {triggerReason === "withdraw"
                ? "Mandatory GDPR consent is required to keep a NutriSync account. Withdrawing consent will immediately and permanently delete your account, health profile, weight/activity logs, and AI insights."
                : "This action cannot be undone. Your NutriSync account, health profile, weight & activity logs, AI insights, and active sessions will be permanently erased."}
            </DialogDescription>
          </DialogHeader>

          <div className="space-y-2 py-2">
            <Label htmlFor="delete-account-confirm-input">
              Type <span className="font-mono font-semibold text-foreground">DELETE</span> to confirm
            </Label>
            <Input
              id="delete-account-confirm-input"
              value={confirmationText}
              onChange={(event) => setConfirmationText(event.target.value)}
              placeholder="DELETE"
              autoComplete="off"
              disabled={isDeleting}
            />
          </div>

          <DialogFooter className="gap-2 sm:gap-2">
            <Button
              type="button"
              variant="outline"
              disabled={isDeleting}
              onClick={() => setDialogOpen(false)}
            >
              Cancel
            </Button>
            <Button
              type="button"
              variant="destructive"
              disabled={confirmationText.trim() !== "DELETE" || isDeleting}
              onClick={() => void handleConfirmDeleteAccount()}
            >
              {isDeleting ? "Deleting..." : "Permanently Delete Account"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </section>
  )
}
