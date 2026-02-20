import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  // Disable font optimization to prevent build failures on servers that cannot reach Google
  optimizeFonts: false,
};

export default nextConfig;
