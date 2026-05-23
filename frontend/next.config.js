const nextConfig = {
  reactStrictMode: true,
  env: {
    NEXT_PUBLIC_API_BASE:
      process.env.NEXT_PUBLIC_API_BASE || "http://localhost:8080",
    NEXT_PUBLIC_API_KEY:
      process.env.NEXT_PUBLIC_API_KEY || "",
  },
};

module.exports = nextConfig;
