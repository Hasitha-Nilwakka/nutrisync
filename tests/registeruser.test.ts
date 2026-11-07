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
