import fs from "node:fs";
import path from "node:path";
import { registerFormSchema } from "../lib/schemas/registerUserSchema";
import { loginUserSchema } from "../lib/schemas/loginUserSchema";
import { describe, expect, it } from "vitest";

const passData = {
  email: "hasitha.nilwakka@gmail.com",
  password: "12345678",
  confirmEmail: "hasitha.nilwakka@gmail.com",
  confirmPassword: "12345678",
};

const failData = {
  email: "hasitha.nilwakka@gmail.com",
  password: "12345678",
  confirmEmail: "hasitha.nilwakka@gmail.com",
  confirmPassword: "1234567",
};

describe("Authentication Zod schema validation", () => {
  it("Validates registration data", () => {
    const parsed = registerFormSchema.safeParse(passData);
    expect(parsed.success).toBe(true);
  });

  it("Invalidates mismatched registration data", () => {
    const parsed = registerFormSchema.safeParse(failData);
    expect(parsed.success).toBe(false);
  });

  it("Validates login credentials", () => {
    const parsed = loginUserSchema.safeParse({
      email: "hasitha.nilwakka@gmail.com",
      password: "12345678",
    });
    expect(parsed.success).toBe(true);
  });

  it("Rejects invalid login credentials", () => {
    const parsed = loginUserSchema.safeParse({
      email: "invalid-email",
      password: "short",
    });
    expect(parsed.success).toBe(false);
  });
});

describe("Registration and email verification UI wiring", () => {
  const repoRoot = path.resolve(__dirname, "..");

  it("uses showAPIResponseToast instead of alert() in Register.tsx", () => {
    const registerSource = fs.readFileSync(
      path.join(repoRoot, "feature/auth/register/Register.tsx"),
      "utf8"
    );
    expect(registerSource).toContain("showAPIResponseToast");
    expect(registerSource).not.toContain("alert(");
  });

  it("wraps VerifyEmailStatus in Suspense and calls /api/auth/verify", () => {
    const pageSource = fs.readFileSync(
      path.join(repoRoot, "app/verify-email/page.tsx"),
      "utf8"
    );
    expect(pageSource).toContain("<Suspense");
    expect(pageSource).toContain("<VerifyEmailStatus");

    const statusSource = fs.readFileSync(
      path.join(repoRoot, "feature/auth/register/VerifyEmailStatus.tsx"),
      "utf8"
    );
    expect(statusSource).toContain('.get("/api/auth/verify"');
    expect(statusSource).toContain("verificationRequestCache");
    expect(statusSource).toContain("verifyEmailTokenOnce");
  });
});
