"use server";

// A guest finds an order by its code and the email it was placed with. The pair is kept in an
// httpOnly cookie for /orders only, so it never goes into a URL; the orders page reads it back.
import { cookies } from "next/headers";
import { redirect } from "next/navigation";
import { lookupOrder } from "@/lib/order-api";

const LOOKUP_COOKIE = "order_lookup";
const LOOKUP_PATH = "/orders";
/** Long enough to read the order and come back to it once or twice. */
const LOOKUP_SECONDS = 15 * 60;

const text = (formData: FormData, name: string) => String(formData.get(name) ?? "").trim();

/** The code and email of the order being looked at, if any. */
export async function lookedUpOrder(): Promise<{ code: string; email: string } | null> {
  const raw = (await cookies()).get(LOOKUP_COOKIE)?.value;
  if (!raw) return null;
  try {
    const { code, email } = JSON.parse(raw) as { code?: unknown; email?: unknown };
    return typeof code === "string" && typeof email === "string" ? { code, email } : null;
  } catch {
    return null;
  }
}

export async function findOrder(formData: FormData) {
  const code = text(formData, "code").toUpperCase();
  const email = text(formData, "email");

  const found = await lookupOrder(code, email);
  if (!found.ok) redirect(`${LOOKUP_PATH}?error=${found.reason}`);

  (await cookies()).set(LOOKUP_COOKIE, JSON.stringify({ code, email }), {
    httpOnly: true,
    sameSite: "lax",
    secure: process.env.NODE_ENV === "production",
    path: LOOKUP_PATH,
    maxAge: LOOKUP_SECONDS,
  });
  redirect(LOOKUP_PATH);
}

/** Back to the empty form, e.g. to look up another order. */
export async function forgetOrder() {
  (await cookies()).delete({ name: LOOKUP_COOKIE, path: LOOKUP_PATH });
  redirect(LOOKUP_PATH);
}
