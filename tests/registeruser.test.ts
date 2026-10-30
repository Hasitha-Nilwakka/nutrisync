import { generateVerificationTokenAndDate, hashpassword } from "../lib/server/auth/registerUser";
import { registerFormSchema } from "../lib/schemas/registerUserSchema";
import { describe, expect, it } from "vitest";

const passData = {
    email: 'hasitha.nilwakka@gmail.com',
    password: '12345678',
    confirmEmail: 'hasitha.nilwakka@gmail.com',
    confirmPassword: '12345678',
    concented: true
}

const failData = {
    email: 'hasitha.nilwakka@gmail.com',
    password: '12345678',
    confirmEmail: 'hasitha.nilwakka@gmail.com',
    confirmPassword: '1234567',
    concented: false
}

describe("New user registration data validation", () => {
  it("Validates data", () => {
    const parsed = registerFormSchema.safeParse(passData)
    expect(parsed.success).toBe(true);
  });
  it("Invalidates data", () => {
    const parsed = registerFormSchema.safeParse(failData)
    expect(parsed.success).toBe(false);
  });
  it("Hashes the password", async () => {
    const password = 'abc@1234'
    const hashed = await hashpassword(password)
    expect(hashed).not.toBe(password)
  });
  it("Generates a verification token", () => {
    const { token } = generateVerificationTokenAndDate()
    expect(token).toMatch(/^[a-f0-9]{64}$/)
  });
  it("Generates a future expiry date", () => {
    const { expiryDate } = generateVerificationTokenAndDate()
    expect(expiryDate.getTime()).toBeGreaterThan(Date.now())
  });
});
