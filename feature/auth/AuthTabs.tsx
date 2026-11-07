// Feature component providing tabbed navigation between LoginForm and RegistrationForm
// on the root route (/), defaulting to the Login tab and rendering Google and GitHub OAuth wireframe buttons in both tabs.
"use client"

import { FieldSeparator } from "@/components/ui/field"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import GithubAuthButton from "@/feature/auth/GithubAuthButton"
import GoogleAuthButton from "@/feature/auth/GoogleAuthButton"
import LoginForm from "@/feature/auth/login/login"
import RegistrationForm from "@/feature/auth/register/Register"

// AuthTabs renders a unified Tabs interface for switching between Login and Register forms,
// including shared "Continue with Google" and "Continue with GitHub" wireframe options in each tab panel.
export default function AuthTabs() {
  return (
    <Tabs defaultValue="login" className="w-full max-w-md">
      <TabsList className="grid w-full grid-cols-2">
        <TabsTrigger value="login">Login</TabsTrigger>
        <TabsTrigger value="register">Register</TabsTrigger>
      </TabsList>

      <TabsContent value="login" className="w-full">
        <div className="flex w-full flex-col gap-4 rounded-lg border p-4 sm:p-6">
          <LoginForm />
          <FieldSeparator>Or</FieldSeparator>
          <div className="flex flex-col gap-2">
            <GoogleAuthButton />
            <GithubAuthButton />
          </div>
        </div>
      </TabsContent>

      <TabsContent value="register" className="w-full">
        <div className="flex w-full flex-col gap-4 rounded-lg border p-4 sm:p-6 [&_form]:border-0">
          <RegistrationForm />
          <FieldSeparator>Or</FieldSeparator>
          <div className="flex flex-col gap-2">
            <GoogleAuthButton />
            <GithubAuthButton />
          </div>
        </div>
      </TabsContent>
    </Tabs>
  )
}
