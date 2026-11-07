// Wireframe button component to initiate Two-Factor Authentication (2FA) setup from the user profile page.
"use client"

import { Button } from "@/components/ui/button"

// EnableTwoFactorButton renders the wireframe trigger button for enabling 2-Factor Authentication.
export default function EnableTwoFactorButton() {
  return (
    <Button type="button" variant="outline">
      Enable 2 Factor Authentication
    </Button>
  )
}
