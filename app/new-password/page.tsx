"use client"
import { showRedirectToast } from "@/components/ui/showToast"
import { useParams, useRouter } from "next/navigation"

export default function ResetPasswordPage() {
    const { token } = useParams()
    const router = useRouter()
    if (!token) {
        showRedirectToast({
            title : 'Unauthorized action',
            message : 'This action seems unauthorized',
            redirectTo : 'login page',
            type : 'error'
        })
        router.replace('/')
        router.refresh()
    }
    return (
        <div>
            You are in reset password page
        </div>
    )
}