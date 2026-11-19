import type { NextConfig } from "next";
import path from "node:path";

const nextConfig: NextConfig = {
  // Copies only the files the server needs into .next/standalone, so the Docker image skips node_modules.
  output: "standalone",
  // One data cache for every replica, kept in Redis (see cache-handler.mjs), instead of one per process.
  cacheHandler: path.resolve("cache-handler.mjs"),
  cacheMaxMemorySize: 0,
};

export default nextConfig;
