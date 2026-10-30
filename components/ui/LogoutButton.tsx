"use client"
import axios from "axios";
import { Button } from "./button";
import { useRouter } from "next/navigation";

//logout button
export default function LogoutButton() {
    const router = useRouter()
    const handleLogout = async() => {
        try {
            await axios.post(
                '/api/auth/logout'
            )

            router.replace('/')
            router.refresh()
        } catch {
            console.error('something went wrong')
        }
    }
    return (
        <Button
            variant={'outline'}
            onClick={ () => handleLogout()}
        >
            Logout
        </Button>
    )
}