import { NextResponse } from "next/server"

const WAKE_TIMEOUT_MS = 65_000

/**
 * GET /api/ai-wake
 * Pings the Python AI service's /health endpoint so free-tier containers
 * that have spun down due to inactivity can wake up before insight generation.
 */
export async function GET() {
  const pyAiBaseUrl = (process.env.PY_AI_URL || "http://localhost:8000").replace(/\/+$/, "")

  try {
    const response = await fetch(`${pyAiBaseUrl}/health`, {
      method: "GET",
      cache: "no-store",
      signal: AbortSignal.timeout(WAKE_TIMEOUT_MS),
    })

    return NextResponse.json({ awake: response.ok }, { status: response.ok ? 200 : 503 })
  } catch {
    return NextResponse.json({ awake: false }, { status: 503 })
  }
}
