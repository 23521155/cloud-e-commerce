import { forwardedFor, GATEWAY_URL } from "@/lib/gateway";
import { authHeader } from "@/lib/session";

export const MAX_PHOTOS = 5;
export const MAX_PHOTO_BYTES = 5 * 1024 * 1024;

const FIELDS = ["title", "author", "year", "edition", "binding", "grade", "faults", "virtues", "asking", "name", "email", "phone"] as const;

export type SellResult = { ok: true; id: number; photos: number } | { ok: false; reason: "invalid" | "too-large" | "busy" | "failed"; detail?: string };

export function photosOf(formData: FormData): File[] {
  return formData.getAll("photos").filter((p): p is File => p instanceof File && p.size > 0);
}

export async function submitSellRequest(formData: FormData): Promise<SellResult> {
  const body = new FormData();
  for (const name of FIELDS) {
    const value = formData.get(name);
    if (typeof value === "string") body.set(name, value);
  }
  for (const photo of photosOf(formData)) body.append("photos", photo, photo.name);

  let res: Response;
  try {
    res = await fetch(`${GATEWAY_URL}/api/sell/requests`, {
      method: "POST",
      body,
      headers: { ...(await authHeader()), ...(await forwardedFor()) },
      cache: "no-store",
    });
  } catch {
    return { ok: false, reason: "failed" };
  }

  if (res.status === 201) {
    const data = (await res.json()) as { id: number; photos: number };
    return { ok: true, id: data.id, photos: data.photos };
  }
  if (res.status === 413) return { ok: false, reason: "too-large" };
  if (res.status === 429) return { ok: false, reason: "busy" };
  if (res.status === 400) {
    const data = (await res.json().catch(() => ({}))) as { error?: string };
    return { ok: false, reason: "invalid", detail: data.error };
  }
  return { ok: false, reason: "failed" };
}

export type SellPhoto = { position: number; contentType: string; sizeBytes: number; width: number | null; height: number | null; processedAt: string | null };

export type SellRequest = {
  id: number;
  title: string;
  author: string;
  year: number | null;
  edition: string | null;
  binding: string;
  grade: string;
  faults: string;
  virtues: string | null;
  askingPriceVnd: number | null;
  sellerName: string;
  sellerEmail: string;
  sellerPhone: string | null;
  status: "PENDING_REVIEW" | "APPROVED" | "REJECTED";
  createdAt: string;
  photos: SellPhoto[];
};

export async function listSellRequests(): Promise<SellRequest[] | null> {
  try {
    const res = await fetch(`${GATEWAY_URL}/api/offers/requests`, { headers: await authHeader(), cache: "no-store" });
    if (!res.ok) return null;
    return ((await res.json()) as { items: SellRequest[] }).items;
  } catch {
    return null;
  }
}

export async function sellPhoto(id: number, position: number): Promise<Response> {
  return fetch(`${GATEWAY_URL}/api/offers/requests/${id}/photos/${position}`, { headers: await authHeader(), cache: "no-store" });
}
