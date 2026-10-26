// Next.js data cache shared through Redis, so every replica of the web app reuses the same
// catalogue answers instead of each keeping its own. Without REDIS_URL (local runs) entries stay
// in this process's memory, like the default cache. A Redis outage only costs the cache: reads
// miss and writes are dropped, and pages still render from the gateway.
import { createClient } from "redis";

const PREFIX = "next-cache:";
const memory = new Map();
let pending;

/** The connected client, or undefined when Redis is not configured or not reachable right now. */
function redis() {
  const url = process.env.REDIS_URL;
  if (!url) return Promise.resolve(undefined);
  pending ??= createClient({ url, disableOfflineQueue: true, socket: { connectTimeout: 2000 } })
    .on("error", (err) => console.warn(`redis cache: ${err.message}`))
    .connect()
    .catch((err) => {
      pending = undefined; // try again on the next call
      console.warn(`redis cache: cannot connect: ${err.message}`);
      return undefined;
    });
  return pending.then((client) => (client?.isReady ? client : undefined));
}

export default class CacheHandler {
  async get(key) {
    const client = await redis();
    if (!client) return memory.get(key) ?? null;
    try {
      const raw = await client.get(PREFIX + key);
      return raw ? JSON.parse(raw) : null;
    } catch {
      return null;
    }
  }

  async set(key, data, ctx) {
    // Only fetch() results are shared: the app caches nothing else, and other kinds hold Buffers.
    if (data?.kind !== "FETCH") return;
    const entry = { value: data, lastModified: Date.now(), tags: ctx?.tags ?? [] };
    const client = await redis();
    if (!client) {
      memory.set(key, entry);
      return;
    }
    try {
      // Next.js decides freshness from lastModified + revalidate; the TTL only clears old keys.
      await client.set(PREFIX + key, JSON.stringify(entry), { EX: Math.max(data.revalidate || 60, 60) * 5 });
    } catch {
      // A lost write only means the next visitor fetches again.
    }
  }

  // The app tags nothing, so entries simply expire.
  async revalidateTag() {}

  resetRequestCache() {}
}
