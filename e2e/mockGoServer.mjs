import http from "node:http"

const PORT = 8088

function makeJwt(state, expSeconds = 3600) {
  const header = Buffer.from(JSON.stringify({ alg: "HS256", typ: "JWT" })).toString("base64url")
  const exp = Math.floor(Date.now() / 1000) + expSeconds
  const payload = Buffer.from(JSON.stringify({ sub: "usr_mock_123", state, exp })).toString("base64url")
  return `${header}.${payload}.mock_sig_${state}`
}

const server = http.createServer((req, res) => {
  const url = new URL(req.url, `http://${req.headers.host || "localhost"}`)
  const path = url.pathname

  // Helper to read body
  let body = ""
  req.on("data", (chunk) => {
    body += chunk
  })

  req.on("end", () => {
    // Parse cookies
    const cookieHeader = req.headers.cookie || ""
    const cookies = Object.fromEntries(
      cookieHeader
        .split(";")
        .map((c) => c.trim().split("="))
        .filter((c) => c.length === 2)
    )
    const token = cookies.access_token

    res.setHeader("Content-Type", "application/json")

    // 1. Session verification
    if (path === "/api/auth/session") {
      if (!token) {
        res.writeHead(401)
        res.end(JSON.stringify({ error: "Unauthorized" }))
        return
      }
      if (token.includes("unconsented")) {
        res.writeHead(200)
        res.end(
          JSON.stringify({
            userId: "usr_unconsented",
            sessionId: "sess_mock_456",
            email: "unconsented@nutrisync.dev",
            consented: false,
            twoFactorEnabled: false,
            hasHealthProfile: false,
          })
        )
        return
      }
      if (token.includes("no_profile")) {
        res.writeHead(200)
        res.end(
          JSON.stringify({
            userId: "usr_no_profile",
            sessionId: "sess_mock_456",
            email: "newuser@nutrisync.dev",
            consented: true,
            twoFactorEnabled: false,
            hasHealthProfile: false,
          })
        )
        return
      }
      if (token.includes("two_factor") || token.includes("2fa")) {
        res.writeHead(200)
        res.end(
          JSON.stringify({
            userId: "usr_2fa",
            sessionId: "sess_mock_456",
            email: "twofactor@nutrisync.dev",
            consented: true,
            twoFactorEnabled: true,
            hasHealthProfile: true,
          })
        )
        return
      }
      // Default valid authenticated session
      res.writeHead(200)
      res.end(
        JSON.stringify({
          userId: "usr_mock_123",
          sessionId: "sess_mock_456",
          email: "tester@nutrisync.dev",
          consented: true,
          twoFactorEnabled: false,
          hasHealthProfile: true,
        })
      )
      return
    }

    // Refresh token handler
    if (path === "/api/auth/refresh") {
      const newToken = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiJ1c3JfbW9ja18xMjMiLCJleHAiOjE5OTk5OTk5OTl9.mock_refreshed_sig"
      res.writeHead(200, {
        "Set-Cookie": [
          `access_token=${newToken}; Path=/; HttpOnly; SameSite=Lax`,
          `refresh_token=mock_refresh_token; Path=/; HttpOnly; SameSite=Lax`,
        ],
      })
      res.end(JSON.stringify({ message: "Token refreshed successfully" }))
      return
    }

    // 2. Health Dashboard
    if (path === "/api/health/dashboard") {
      res.writeHead(200)
      res.end(
        JSON.stringify({
          healthProfile: {
            age: 29,
            gender: "female",
            heightCm: 168,
            weightKg: 64,
            targetWeightKg: 60,
            preferredWeightUnit: "kg",
            preferredHeightUnit: "cm",
            bmi: 22.7,
            bmiClassification: "Normal Weight",
            bmr: 1420,
            tdee: 1950,
            occupationType: "desk_job",
            activityLevel: "moderate",
            targetActivityLevel: "high",
            primaryGoal: "fat_loss",
            targetWeeklyDays: 4,
            weeklyActivityDays: 4,
            exerciseTypes: ["running", "yoga", "swimming"],
            sessionDuration: 45,
            fitnessLevel: "intermediate",
            preferredEnvironment: "outdoors",
            preferredTimeOfDay: "morning",
            enduranceMinutes: 30,
            pushupsCount: 20,
            dietaryPreferences: ["high_protein"],
            dietaryRestrictions: ["gluten_free"],
            medicalConditions: "None",
          },
          latestWellness: {
            id: "snp_123",
            bmiScore: 88,
            activityScore: 82,
            progressScore: 85,
            habitsScore: 80,
            wellnessScore: 84.1,
            calculationDate: new Date().toISOString(),
          },
          weightHistory: [
            {
              id: "w_1",
              weightKg: 64.0,
              weightUnit: "kg",
              loggedAt: new Date().toISOString(),
            },
          ],
          activityHistory: [
            {
              id: "a_1",
              activityType: "running",
              durationMinutes: 30,
              intensity: "moderate",
              caloriesBurned: 280,
              loggedAt: new Date().toISOString(),
            },
          ],
          milestones: [
            {
              id: "m_1",
              category: "weight",
              title: "First 2kg Shed",
              description: "Down 2.5 kg from baseline.",
              achieved: true,
              achievedAt: new Date().toISOString(),
            },
          ],
          aiInsights: {
            dietaryInsights: "Ensure adequate protein distribution around workouts.",
            exerciseRecommendations: "Solid aerobic base; introduce intervals.",
            clinicalGuidance: "Maintain proper hydration.",
            generatedAt: new Date().toISOString(),
            modelUsed: "nutrisync-llama-3.2-1b",
            status: "active",
          },
        })
      )
      return
    }

    // 3. Login
    if (path === "/api/auth/login") {
      let parsed = {}
      try {
        parsed = JSON.parse(body || "{}")
      } catch {}
      if (parsed.email && parsed.email.includes("2fa")) {
        res.writeHead(200)
        res.end(JSON.stringify({ require2FA: true, ticket: "mock_2fa_ticket" }))
        return
      }
      res.setHeader("Set-Cookie", `access_token=${makeJwt("authenticated")}; Path=/; HttpOnly; SameSite=Lax`)
      res.writeHead(200)
      res.end(JSON.stringify({ message: "Login successful" }))
      return
    }

    // 4. 2FA Login
    if (path === "/api/auth/2fa/login") {
      let parsed = {}
      try {
        parsed = JSON.parse(body || "{}")
      } catch {}
      if (parsed.code === "123456") {
        res.setHeader("Set-Cookie", `access_token=${makeJwt("authenticated")}; Path=/; HttpOnly; SameSite=Lax`)
        res.writeHead(200)
        res.end(JSON.stringify({ message: "2FA login successful" }))
        return
      }
      res.writeHead(400)
      res.end(JSON.stringify({ message: "Invalid or expired 2FA code." }))
      return
    }

    // 5. Register
    if (path === "/api/auth/register") {
      res.writeHead(201)
      res.end(
        JSON.stringify({
          type: "success",
          title: "Registration Successful",
          message: "Account created. Please check your email to verify your address.",
        })
      )
      return
    }

    // 6. Consent
    if (path === "/api/auth/consent") {
      res.setHeader("Set-Cookie", `access_token=${makeJwt("no_profile")}; Path=/; HttpOnly; SameSite=Lax`)
      res.writeHead(200)
      res.end(JSON.stringify({ message: "Consent updated successfully" }))
      return
    }

    // 7. Health Profile
    if (path === "/api/health/profile") {
      res.setHeader("Set-Cookie", `access_token=${makeJwt("authenticated")}; Path=/; HttpOnly; SameSite=Lax`)
      res.writeHead(200)
      res.end(
        JSON.stringify({
          type: "success",
          title: "Profile Saved",
          message: "Health profile saved successfully.",
        })
      )
      return
    }

    // 8. Log Weight
    if (path === "/api/health/weight") {
      res.writeHead(201)
      res.end(
        JSON.stringify({
          type: "success",
          title: "Weight Logged",
          message: "Weight entry recorded successfully.",
        })
      )
      return
    }

    // 9. Log Activity
    if (path === "/api/health/activity") {
      res.writeHead(201)
      res.end(
        JSON.stringify({
          type: "success",
          title: "Activity Logged",
          message: "Activity entry recorded successfully.",
        })
      )
      return
    }

    // 10. AI Regenerate
    if (path === "/api/health/ai/regenerate" || path === "/api/health/insights/regenerate") {
      res.writeHead(200)
      res.end(
        JSON.stringify({
          latestInsights: {
            id: "ins_124",
            userId: "usr_mock_123",
            version: 2,
            primaryGoalSnapshot: "fat_loss",
            promptStrategy: "few_shot",
            healthStatusAnalysis: "Updated analysis: Cardiovascular adaptation continues smoothly.",
            progressEvaluation: "Weight trajectory demonstrates sustainable weekly decrement towards target.",
            recommendationsJson: JSON.stringify([
              {
                title: "Target Aerobic Zone",
                category: "exercise",
                priority: "high",
                action: "Maintain 130-150 bpm during 45-min cardio blocks.",
                expectedImpact: "Optimizes fat oxidation without muscle catabolism.",
              },
            ]),
            weeklySummary: "Excellent consistency across 4 scheduled workouts.",
            monthlySummary: "Down 2.5 kg over the past month.",
            createdAt: new Date().toISOString(),
          },
          insightHistory: [],
          aiServiceAvailable: true,
          isCachedFallback: false,
        })
      )
      return
    }

    // 11. Export
    if (path === "/api/health/export") {
      res.setHeader("Content-Disposition", 'attachment; filename="nutrisync-health-data.json"')
      res.writeHead(200)
      res.end(JSON.stringify({ exportedAt: new Date().toISOString() }))
      return
    }

    // 12. Account Deletion
    if (path === "/api/auth/account" && req.method === "DELETE") {
      res.setHeader(
        "Set-Cookie",
        "access_token=; Path=/; Expires=Thu, 01 Jan 1970 00:00:00 GMT; HttpOnly; SameSite=Lax"
      )
      res.writeHead(200)
      res.end(JSON.stringify({ message: "Account deleted successfully" }))
      return
    }

    // 13. Password reset request
    if (path === "/api/auth/passwordreset") {
      res.writeHead(200)
      res.end(
        JSON.stringify({
          type: "success",
          title: "Reset Email Sent",
          message: "If that email is registered, we have sent a reset link.",
        })
      )
      return
    }

    // 14. Password reset confirm
    if (path === "/api/auth/passwordreset/confirm") {
      res.writeHead(200)
      res.end(
        JSON.stringify({
          type: "success",
          title: "Password Reset",
          message: "Your password has been successfully updated.",
        })
      )
      return
    }

    // 15. 2FA setup
    if (path === "/api/auth/2fa/setup") {
      res.writeHead(200)
      res.end(
        JSON.stringify({
          secret: "JBSWY3DPEHPK3PXP",
          otpauthUrl: "otpauth://totp/NutriSync:test@nutrisync.dev?secret=JBSWY3DPEHPK3PXP&issuer=NutriSync",
          qrCodeDataUrl: "data:image/svg+xml;utf8,<svg xmlns='http://www.w3.org/2000/svg' width='100' height='100'><rect width='100' height='100' fill='black'/></svg>",
        })
      )
      return
    }

    // 16. 2FA enable
    if (path === "/api/auth/2fa/enable") {
      res.writeHead(200)
      res.end(
        JSON.stringify({
          type: "success",
          title: "2FA Enabled",
          message: "Two-factor authentication is now active.",
        })
      )
      return
    }

    // 17. 2FA disable
    if (path === "/api/auth/2fa/disable") {
      res.writeHead(200)
      res.end(
        JSON.stringify({
          type: "success",
          title: "2FA Disabled",
          message: "Two-factor authentication has been disabled.",
        })
      )
      return
    }

    // 18. Logout
    if (path === "/api/auth/logout") {
      res.setHeader(
        "Set-Cookie",
        "access_token=; Path=/; Expires=Thu, 01 Jan 1970 00:00:00 GMT; HttpOnly; SameSite=Lax"
      )
      res.writeHead(200)
      res.end(JSON.stringify({ message: "Logged out" }))
      return
    }

    // Default 404
    res.writeHead(404)
    res.end(JSON.stringify({ error: "Not found" }))
  })
})

server.listen(PORT, "127.0.0.1", () => {
  console.log(`[MockGoServer] Listening on http://127.0.0.1:${PORT}`)
})
