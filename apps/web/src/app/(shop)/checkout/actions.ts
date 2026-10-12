"use server";

import { redirect } from "next/navigation";
import { MAX_BASKET } from "@/lib/basket";
import { placeOrder as submitOrder } from "@/lib/order-api";
import { getPaymentMethod } from "@/lib/payment";

/** What the checkout form shows after a refused order. Empty while nothing went wrong. */
export type CheckoutState = {
  error?: string;
  /** Slugs of the copies someone else bought first; the form shows their titles. */
  unavailable?: string[];
};

const text = (formData: FormData, name: string) => String(formData.get(name) ?? "").trim();

/** Form data stays in the POST body, never the URL. */
export async function placeOrder(_prev: CheckoutState, formData: FormData): Promise<CheckoutState> {
  const method = getPaymentMethod(formData.get("payment"));
  if (!method) return { error: "Choose how you would like to pay." };

  // Only cash on delivery is placed for real. Online methods keep the mock provider page
  // (/checkout/pay) until a payment gateway is integrated.
  if (method.id !== "cod") {
    redirect(method.flow === "gateway" ? `/checkout/pay?method=${method.id}` : `/checkout/confirmed?method=${method.id}`);
  }

  // The basket lives in the browser, so its slugs come with the form. Only the slugs: the order
  // service reads every price and checks every copy itself.
  const slugs = [...new Set(formData.getAll("slug").map(String))];
  if (slugs.length === 0) return { error: "Your basket has no copies that can still be bought." };
  if (slugs.length > MAX_BASKET) return { error: `One order holds ${MAX_BASKET} copies at most.` };

  const postal = text(formData, "postal");
  // CheckoutForm adds the key; a form posted before the page hydrated has none
  const key = text(formData, "idempotencyKey") || crypto.randomUUID();
  const result = await submitOrder(key, {
    slugs,
    paymentMethod: "cod",
    email: text(formData, "email"),
    phone: text(formData, "phone"),
    fullName: text(formData, "name"),
    address: text(formData, "address"),
    city: text(formData, "city"),
    postalCode: postal || null,
    country: text(formData, "country"),
  });

  if (result.ok) redirect(`/checkout/confirmed?method=cod&code=${encodeURIComponent(result.order.code)}`);

  switch (result.reason) {
    case "unavailable":
      return { error: "Another reader bought these copies a moment ago, so nothing was ordered:", unavailable: result.slugs };
    case "invalid":
      return { error: `Please check your details: ${result.message}.` };
    default:
      return { error: "The order could not be placed right now. Nothing was charged; please try again in a moment." };
  }
}
