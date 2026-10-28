import { loginUserSchema } from "../lib/schemas/loginUserSchema";
import { describe, expect, it } from "vitest";

describe("Test user login zod validation", () => {
    it("It passes valid email and password", () => {
        const data = {
            email : 'example@example.com',
            password : 'abc@1234567'
        }
        const parsed = loginUserSchema.safeParse(data)
        expect(parsed.success).toBe(true);
    });
    it("It invalidates invalid email", () => {
        const data = {
            email : 'exampleexample.com',
            password : 'abc@1234567'
        }
        const parsed = loginUserSchema.safeParse(data)
        expect(parsed.success).toBe(false);
    });
    it("It invalidates missing email", () => {
        const data = {
            email : '',
            password : 'abc@1234567'
        }
        const parsed = loginUserSchema.safeParse(data)
        expect(parsed.success).toBe(false);
    });
    it("It invalidates passwords less than 8 chars", () => {
        const data = {
            email : 'example@example.com',
            password : 'abc@1'
        }
        const parsed = loginUserSchema.safeParse(data)
        expect(parsed.success).toBe(false);
    });
    it("It invalidates missing password", () => {
        const data = {
            email : 'example@example.com',
            password : ''
        }
        const parsed = loginUserSchema.safeParse(data)
        expect(parsed.success).toBe(false);
    });
});