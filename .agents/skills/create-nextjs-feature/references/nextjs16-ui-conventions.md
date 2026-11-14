# Next.js 16, Zod v4 & UI Architecture Conventions (`/` Root)

## 1. Folder Ownership & Separation of Concerns (`AGENTS.md` §3.1)

- **`app/` (Routing Only):**
  - Route folders (`app/dashboard/`, `app/profile/`, `app/new-password/`) must remain thin (`page.tsx`, `layout.tsx`, `loading.tsx`).
  - Use default exports only for Next.js entry files (`page.tsx`, `layout.tsx`, `next.config.ts`). Use named exports everywhere else.
- **`feature/` (Domain UI & Flows):**
  - All domain forms, multi-step wizards, charts, and interactive panels belong in `feature/<domain>/` (`feature/auth/`, `feature/dashboard/`, `feature/health-profile/`, `feature/profile/`).
- **`components/ui/` (Foundational Primitives):**
  - Contains reusable Shadcn/ui primitives (`button.tsx`, `dialog.tsx`, `input.tsx`, `select.tsx`, `tabs.tsx`, `showToast.ts`). Do not place business logic inside `components/ui/`.
- **`lib/schemas/` (Zod v4 Contracts):**
  - Central location for all form and client trust-boundary schemas (`healthProfileSchema.ts`, `loginUserSchema.ts`, `registerUserSchema.ts`, `passwordChangeSchema.ts`).

---

## 2. API Communication & Security Rules (`AGENTS.md` §6, §8, §9)

- **Relative `/api/*` Rewrites Only:**
  - All client-side `axios` calls must target relative paths (`/api/auth/*`, `/api/health/*`, `/api/go/*`) configured in [`next.config.ts`](../../../../next.config.ts).
  - Never hardcode `http://localhost:8080` or `http://localhost:8000` in client components.
- **No Direct Database Access:**
  - Next.js must never query PostgreSQL directly or import from `src/generated/prisma/`. All data persistence goes through `services/go-api`.
- **Server-Side Authentication Gate:**
  - Protected pages in `app/` must invoke `requireAuthentication` from [`lib/auth/requireAuthentication.ts`](../../../../lib/auth/requireAuthentication.ts) before rendering user-scoped views.
