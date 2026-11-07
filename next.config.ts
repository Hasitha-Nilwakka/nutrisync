import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  /* config options here */
  experimental: {
    agentFeedback: true,
  },
  output : "standalone",
  cacheComponents: true,
  partialPrefetching: true,
  turbopack: {
    rules: {
      "*.css": {
        loaders: ["@tailwindcss/turbopack"],
        as: "*.css",
      },
    },
  },
  async rewrites() {
    const goApiBaseUrl = process.env.GO_API_URL || "http://localhost:8080";
    return {
      beforeFiles: [
        {
          source: "/api/go/:path*",
          destination: `${goApiBaseUrl}/api/auth/:path*`,
        },
        {
          source: "/api/auth/register",
          destination: `${goApiBaseUrl}/api/auth/register`,
        },
        {
          source: "/api/auth/verify",
          destination: `${goApiBaseUrl}/api/auth/verify`,
        },
        {
          source: "/api/auth/login",
          destination: `${goApiBaseUrl}/api/auth/login`,
        },
        {
          source: "/api/auth/refresh",
          destination: `${goApiBaseUrl}/api/auth/refresh`,
        },
        {
          source: "/api/auth/logout",
          destination: `${goApiBaseUrl}/api/auth/logout`,
        },
        {
          source: "/api/auth/session",
          destination: `${goApiBaseUrl}/api/auth/session`,
        },
        {
          source: "/api/auth/oauth/google",
          destination: `${goApiBaseUrl}/api/auth/oauth/google`,
        },
        {
          source: "/api/auth/oauth/google/callback",
          destination: `${goApiBaseUrl}/api/auth/oauth/google/callback`,
        },
      ],
    };
  },
};

export default nextConfig;
