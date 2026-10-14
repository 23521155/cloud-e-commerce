import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  // Copies only the files the server needs into .next/standalone, so the Docker image skips node_modules.
  output: "standalone",
};

export default nextConfig;
