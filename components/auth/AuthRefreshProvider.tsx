"use client"

import { useEffect } from "react"
import { installAxiosRefreshInterceptor } from "@/lib/api/setupAxiosRefreshInterceptor"

export function AuthRefreshProvider() {
  useEffect(() => {
    installAxiosRefreshInterceptor()
  }, [])

  return null
}
