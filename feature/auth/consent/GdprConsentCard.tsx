// Client component rendering the mandatory GDPR & Health Data Processing Consent Gate for OAuth users
// who have not yet accepted data processing terms (session.consented === false).
"use client"

import { useState } from "react"
import axios from "axios"
import { useRouter } from "next/navigation"
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
import { Spinner } from "@/components/ui/spinner"

interface GdprConsentCardProps {
  email?: string
}

// GdprConsentCard blocks access to health profile & dashboard views until the user explicitly accepts
// GDPR health data processing terms or declines (which deletes their unconsented account).
export default function GdprConsentCard({ email }: GdprConsentCardProps) {
  const router = useRouter()
  const [isConsentChecked, setIsConsentChecked] = useState(false)
  const [pendingAction, setPendingAction] = useState<"accept" | "decline" | null>(null)
  const [errorMessage, setErrorMessage] = useState<string | null>(null)

  const handleAcceptConsent = async () => {
    if (!isConsentChecked || pendingAction !== null) {
      return
    }

    setPendingAction("accept")
    setErrorMessage(null)
    try {
      await axios.post("/api/auth/consent", { consented: true })
      router.replace("/dashboard")
      router.refresh()
    } catch (error) {
      const message = axios.isAxiosError(error)
        ? error.response?.data?.message || "Failed to record consent. Please try again."
        : "Failed to record consent. Please try again."
      setErrorMessage(message)
      setPendingAction(null)
    }
  }

  const handleDeclineAndDeleteAccount = async () => {
    if (pendingAction !== null) {
      return
    }

    setPendingAction("decline")
    setErrorMessage(null)
    try {
      await axios.post("/api/auth/consent", { consented: false })
      router.replace("/")
      router.refresh()
    } catch (error) {
      const message = axios.isAxiosError(error)
        ? error.response?.data?.message || "Failed to remove account. Please try again."
        : "Failed to remove account. Please try again."
      setErrorMessage(message)
      setPendingAction(null)
    }
  }

  return (
    <div className="mx-auto flex min-h-[calc(100dvh-8rem)] w-full max-w-xl items-center justify-center px-4 py-8">
      <div className="w-full rounded-lg border p-6">
        <FieldGroup>
          <FieldSet>
            <FieldLegend>Data Privacy &amp; Health Data Consent</FieldLegend>
            <FieldDescription>
              {email
                ? `Signed in as ${email}. Before continuing to NutriSync, please review and confirm how your data is processed.`
                : "Before continuing to NutriSync, please review and confirm how your data is processed."}
            </FieldDescription>

            <FieldSeparator />

            <FieldGroup>
              <FieldDescription>
                <strong>1. Health &amp; Lifestyle Data Collected:</strong> NutriSync collects and processes your profile
                details, physical metrics (age, gender, height, weight), lifestyle indicators (activity level, dietary
                preferences, restrictions), and fitness goals to calculate wellness indicators such as BMI and progress
                scores.
              </FieldDescription>
              <FieldDescription>
                <strong>2. AI Insight Generation &amp; Anonymization:</strong> Personal identifiers (such as your email
                address and account ID) are stripped and anonymized before your health metrics are processed to generate
                personalized AI wellness insights and recommendations.
              </FieldDescription>
              <FieldDescription>
                <strong>3. Your GDPR Rights:</strong> You may export a copy of your data, update your privacy settings,
                or withdraw your consent and permanently delete your account at any time.
              </FieldDescription>
            </FieldGroup>

            <FieldSeparator />

            <Field orientation="horizontal">
              <Checkbox
                id="gdpr-consent-checkbox"
                checked={isConsentChecked}
                disabled={pendingAction !== null}
                onCheckedChange={(checked) => setIsConsentChecked(checked === true)}
              />
              <FieldLabel htmlFor="gdpr-consent-checkbox" className="font-normal">
                I consent to the processing of my personal and health-related data
              </FieldLabel>
            </Field>

            {errorMessage && <FieldError>{errorMessage}</FieldError>}

            <Field orientation="horizontal" className="justify-between gap-3 pt-2">
              <Button
                type="button"
                disabled={!isConsentChecked || pendingAction !== null}
                onClick={handleAcceptConsent}
              >
                {pendingAction === "accept" && <Spinner />}
                {pendingAction === "accept" ? "Saving..." : "Accept & Continue"}
              </Button>
              <Button
                type="button"
                variant="outline"
                disabled={pendingAction !== null}
                onClick={handleDeclineAndDeleteAccount}
              >
                {pendingAction === "decline" && <Spinner />}
                {pendingAction === "decline" ? "Removing..." : "Decline & Delete Account"}
              </Button>
            </Field>
          </FieldSet>
        </FieldGroup>
      </div>
    </div>
  )
}
