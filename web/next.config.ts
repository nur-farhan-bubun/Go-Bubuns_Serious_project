import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  images: {
    remotePatterns: [
      {
        protocol: "https",
        hostname: "images.unsplash.com",
      },
      {
        protocol: "https",
        hostname: "randomuser.me",
      },
    ],
  },
  reactStrictMode: true,
  // Proxy API requests to the backend (API gateway on port 8080)
  // to avoid CORS issues when the frontend (port 3000) calls the API (port 8080).
  async rewrites() {
    return [
      {
        source: "/v1/:path*",
        destination: "http://localhost:8080/v1/:path*",
      },
      {
        source: "/ws",
        destination: "http://localhost:8080/ws",
      },
    ];
  },
};

export default nextConfig;
