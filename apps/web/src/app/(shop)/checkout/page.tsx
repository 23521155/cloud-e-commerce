import type { Metadata } from "next";
import type { CheckoutPrefill } from "@/components/checkout/Checkout";
import { CheckoutView } from "@/components/views/CheckoutView";
import { getSession } from "@/lib/session";
import { me } from "@/lib/user-api";

export const metadata: Metadata = {
  title: "Checkout — Marginalleya",
  description: "Delivery and payment for the copies in your basket.",
};

/** The details a signed-in reader keeps on the account, to start the form with; none for a guest. */
async function prefill(): Promise<CheckoutPrefill | undefined> {
  const session = await getSession();
  if (!session) return undefined;

  // Checkout still works if the account cannot be read: the reader types the details in.
  const result = await me(session.token);
  if (!result.ok) return undefined;

  const { account } = result;
  return {
    email: account.email,
    phone: account.phone,
    name: account.name,
    address: account.address,
    city: account.city,
    postal: account.postalCode,
    country: account.country,
  };
}

/** The basket is in the browser, so the view reads it; the reader's details come from here. */
export default async function CheckoutPage() {
  return <CheckoutView prefill={await prefill()} />;
}
