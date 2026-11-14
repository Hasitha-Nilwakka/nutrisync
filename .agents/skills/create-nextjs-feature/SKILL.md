---
name: create-nextjs-feature
description: >-
  Build and verify Next.js 16 App Router pages, feature-scoped UI modules in feature/,
  and Zod v4 validation schemas in lib/schemas/ with Vitest unit tests.
  Use whenever creating or updating frontend routes, forms, schemas, or UI components.
---

# Create Next.js Feature & Schema (`create-nextjs-feature`)

Follow this runbook when creating or modifying Frontend routes (`app/`), feature modules (`feature/`), and Zod v4 validation schemas (`lib/schemas/`) for **NutriSync (`numbers-dont-lie`)**.

---

## 1. Progressive Context & Boilerplate

Before writing code, consult these bundled resources as needed:
- **Next.js 16 & UI Conventions:** Read [`references/nextjs16-ui-conventions.md`](./references/nextjs16-ui-conventions.md) for App Router rules, Shadcn/ui usage, `/api/*` rewrite conventions, and session checks.
- **Zod v4 Schema Starter Template:** Copy and adapt [`assets/schema.template.ts`](./assets/schema.template.ts).
- **Vitest Schema Test Starter Template:** Copy and adapt [`assets/schema.test.template.ts`](./assets/schema.test.template.ts).

---

## 2. Step-by-Step Implementation Flow

1. **Check Next.js 16 & Library Docs (`context7` MCP & Local Docs):**
   - Remember that this project runs **Next.js `16.4.2`**, **React `19.3.0`**, **Tailwind CSS `v4`**, and **Zod `v4.6.7`**.
   - Check `node_modules/next/dist/docs/` or query `context7` MCP before using App Router APIs (`cookies()`, `headers()`, `params`, `searchParams` are asynchronous in Next.js 16).
2. **Define Trust-Boundary Validation Schema (`lib/schemas/`):**
   - Adapt [`assets/schema.template.ts`](./assets/schema.template.ts) using Zod v4 and export both the schema and `z.infer<typeof ...>` TypeScript type.
   - Keep field names and physiological bounds aligned with the Go handler payload in `services/go-api/handlers/`.
3. **Build Feature-Scoped UI Module (`feature/<domain>/`):**
   - Place domain components, wizards, and form flows inside `feature/<domain>/` (e.g., `feature/auth/`, `feature/dashboard/`, `feature/health-profile/`, `feature/profile/`).
   - Reuse foundational primitives from `components/ui/` (Shadcn/ui) and `components/ui/showToast.ts`.
   - Use **React Hook Form** + `@hookform/resolvers/zod` for form state and **Axios** with relative `/api/*` paths (rewritten via `next.config.ts`)—**never** hardcode `http://localhost:8080` or expose server secrets in client components.
4. **Wire Thin App Router Page (`app/<route>/page.tsx`):**
   - Keep `app/<route>/page.tsx` thin: verify server-side authentication via `lib/auth/requireAuthentication.ts` (for protected routes) and render the `feature/` component.
5. **Add Vitest Unit Tests (`tests/<feature>.test.ts`):**
   - Adapt [`assets/schema.test.template.ts`](./assets/schema.test.template.ts) to test valid inputs and invalid boundary rejections.

---

## 3. Deterministic Verification

Run the bundled verification script to execute TypeScript typechecking (`tsc --noEmit`), ESLint, and Vitest unit tests:

```bash
./.agents/skills/create-nextjs-feature/scripts/verify-frontend.sh
```
