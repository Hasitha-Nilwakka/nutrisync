// GdprExportButton fetches GET /api/health/export and downloads the user's full GDPR health data JSON file.
"use client"

import { useState } from "react"
import axios from "axios"
import { Download, Loader2 } from "lucide-react"
import { Button } from "@/components/ui/button"
import { showAPIResponseToast } from "@/components/ui/showToast"
import type { GDPRHealthDataExportPayload } from "@/lib/schemas/dashboardSchema"

export function GdprExportButton() {
  const [isExporting, setIsExporting] = useState(false)

  const handleExport = async () => {
    setIsExporting(true)
    try {
      const response = await axios.get<GDPRHealthDataExportPayload>("/api/health/export", {
        withCredentials: true,
      })

      const jsonText = JSON.stringify(response.data, null, 2)
      const blob = new Blob([jsonText], { type: "application/json" })
      const objectUrl = URL.createObjectURL(blob)

      const dateStamp = new Date().toISOString().slice(0, 10)
      const anchor = document.createElement("a")
      anchor.href = objectUrl
      anchor.download = `nutrisync-health-export-${dateStamp}.json`
      document.body.appendChild(anchor)
      anchor.click()
      anchor.remove()
      URL.revokeObjectURL(objectUrl)

      showAPIResponseToast({
        type: "success",
        title: "Health Data Exported",
        message: "Your GDPR health data JSON archive has been downloaded.",
      })
    } catch {
      showAPIResponseToast({
        type: "error",
        title: "Export Failed",
        message: "Unable to export your health data right now. Please try again.",
      })
    } finally {
      setIsExporting(false)
    }
  }

  return (
    <Button
      type="button"
      variant="outline"
      size="sm"
      disabled={isExporting}
      onClick={handleExport}
    >
      {isExporting ? <Loader2 className="size-4 animate-spin" /> : <Download className="size-4" />}
      <span>Export JSON</span>
    </Button>
  )
}
