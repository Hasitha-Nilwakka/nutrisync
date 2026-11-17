// Feature component providing tabbed navigation between LoginForm and RegistrationForm
// on the root route (/), defaulting to the Login tab and rendering Google and GitHub OAuth options in both tabs.
"use client"

import { FieldSeparator } from "@/components/ui/field"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import GithubAuthButton from "@/feature/auth/GithubAuthButton"
import GoogleAuthButton from "@/feature/auth/GoogleAuthButton"
import LoginForm from "@/feature/auth/login/login"
import RegistrationForm from "@/feature/auth/register/Register"

// Shared OAuth provider buttons with a temporary-disabled notice for free-tier hosting.
function OAuthProvidersSection() {
  return (
    <div className="pt-2">
      <FieldSeparator>Or continue with</FieldSeparator>
      <div className="mt-5 flex flex-col gap-2.5">
        <GoogleAuthButton disabled />
        <GithubAuthButton disabled />
      </div>
      <p className="mt-3 text-center text-xs text-muted-foreground">
        Google &amp; GitHub OAuth sign-in is temporarily disabled due to hosting limitations. Please use email and password.
      </p>
    </div>
  )
}

// AuthTabs renders a unified card with Tabs for switching between Login and Register forms,
// including shared "Continue with Google" and "Continue with GitHub" options in each tab panel.
export default function AuthTabs() {
  return (
    <div className="w-full max-w-md overflow-hidden rounded-2xl border border-border/90 bg-card/95 shadow-lg backdrop-blur-sm">
      <div
        aria-hidden="true"
        className="h-1.5 w-full bg-linear-to-r from-[#0B74B8] via-[#18A999] to-[#43B047]"
      />
      <div className="p-5 sm:p-6">
        <Tabs defaultValue="login" className="w-full gap-5">
          <TabsList className="grid h-auto w-full grid-cols-2 rounded-xl border-border/60 bg-muted/80 p-1">
            <TabsTrigger
              value="login"
              className="h-9 rounded-lg border border-transparent py-0 text-muted-foreground data-active:border-border/60 data-active:bg-card data-active:text-primary data-active:shadow-xs"
            >
              Login
            </TabsTrigger>
            <TabsTrigger
              value="register"
              className="h-9 rounded-lg border border-transparent py-0 text-muted-foreground data-active:border-border/60 data-active:bg-card data-active:text-primary data-active:shadow-xs"
            >
              Register
            </TabsTrigger>
          </TabsList>

          <TabsContent value="login" className="w-full space-y-5">
            <LoginForm />
            <OAuthProvidersSection />
          </TabsContent>

          <TabsContent value="register" className="w-full space-y-5">
            <RegistrationForm />
            <OAuthProvidersSection />
          </TabsContent>
        </Tabs>
      </div>
    </div>
  )
}
