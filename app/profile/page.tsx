// Profile route (/profile) page rendering the user profile wireframe with the Enable 2 Factor Authentication button.
import EnableTwoFactorButton from "@/feature/profile/EnableTwoFactorButton"

export default function ProfilePage() {
  return (
    <div className="mx-auto flex min-h-[calc(100dvh-8rem)] w-full max-w-4xl flex-col items-start justify-start gap-6 px-4 py-8 sm:px-6 lg:px-8">
      <EnableTwoFactorButton />
    </div>
  )
}
