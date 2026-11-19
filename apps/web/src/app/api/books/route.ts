import { MAX_BASKET } from "@/lib/basket";
import { getBooks } from "@/lib/catalogue";

/**
 * GET /api/books?slugs=a,b,c: the books of a basket, for pages that read the basket in the browser.
 * Proxies the catalogue so the gateway stays server-side. Books no longer for sale are included,
 * with available: false; unknown slugs are left out.
 */
export async function GET(request: Request) {
  const slugs = (new URL(request.url).searchParams.get("slugs") ?? "")
    .split(",")
    .map((s) => s.trim())
    .filter(Boolean);

  if (slugs.length > MAX_BASKET) return Response.json({ error: `at most ${MAX_BASKET} books` }, { status: 400 });

  try {
    return Response.json(await getBooks(slugs));
  } catch {
    return Response.json({ error: "catalogue unavailable" }, { status: 502 });
  }
}
