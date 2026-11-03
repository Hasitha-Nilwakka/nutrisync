
import { describe, it, expect } from "vitest";
import {
  createAccessToekn,
  verifyAccessToken,
  generateRefreshTokenSecret,
  hashRefreshTokenSecret,
} from "../lib/server/auth/tokens";

describe("Access Token Functions", () => {

  describe("createAccessToekn", () => {

    it("should create a valid JWT string", async () => {
      const token = await createAccessToekn(
        "user-123",
        "session-456"
      );

      expect(token).toBeTypeOf("string");
      expect(token.split(".")).toHaveLength(3);
    });

    it("should include the correct userId and sessionId", async () => {
      const token = await createAccessToekn(
        "user-123",
        "session-456"
      );

      const payload = await verifyAccessToken(token);

      expect(payload.sub).toBe("user-123");
      expect(payload.sid).toBe("session-456");
    });

  });


  describe("verifyAccessToken", () => {

    it("should successfully verify a valid token", async () => {
      const token = await createAccessToekn(
        "user-123",
        "session-456"
      );

      const payload = await verifyAccessToken(token);

      expect(payload).toBeDefined();
      expect(payload.sub).toBe("user-123");
      expect(payload.sid).toBe("session-456");
    });

    it("should reject a tampered token", async () => {
      const token = await createAccessToekn(
        "user-123",
        "session-456"
      );

      // Modify the token
      const tamperedToken = token.slice(0, -1) + "x";

      await expect(
        verifyAccessToken(tamperedToken)
      ).rejects.toThrow();
    });

    it("should reject an invalid token", async () => {
      await expect(
        verifyAccessToken("this-is-not-a-jwt")
      ).rejects.toThrow();
    });

  });


  describe("generateRefreshTokenSecret", () => {

    it("should generate a string", () => {
      const secret = generateRefreshTokenSecret();

      expect(secret).toBeTypeOf("string");
    });

    it("should generate a 32-byte secret encoded as base64url", () => {
      const secret = generateRefreshTokenSecret();

      // 32 bytes encoded as base64url produces 43 characters
      expect(secret).toMatch(/^[A-Za-z0-9_-]{43}$/);
    });

    it("should generate different secrets each time", () => {
      const secret1 = generateRefreshTokenSecret();
      const secret2 = generateRefreshTokenSecret();

      expect(secret1).not.toBe(secret2);
    });

  });


  describe("hashRefreshTokenSecret", () => {

    it("should hash a refresh token secret", () => {
      const hash = hashRefreshTokenSecret("my-secret");

      expect(hash).toBeTypeOf("string");
      expect(hash).toHaveLength(64);
    });

    it("should generate the same hash for the same secret", () => {
      const hash1 = hashRefreshTokenSecret("my-secret");
      const hash2 = hashRefreshTokenSecret("my-secret");

      expect(hash1).toBe(hash2);
    });

    it("should generate different hashes for different secrets", () => {
      const hash1 = hashRefreshTokenSecret("secret-one");
      const hash2 = hashRefreshTokenSecret("secret-two");

      expect(hash1).not.toBe(hash2);
    });

  });

});
