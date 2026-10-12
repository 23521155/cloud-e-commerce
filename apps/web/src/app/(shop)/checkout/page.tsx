import type { Metadata } from "next";
import { CheckoutView } from "@/components/views/CheckoutView";

export const metadata: Metadata = {
  title: "Checkout — Marginalleya",
  description: "Delivery and payment for the copies in your basket.",
};

/** The basket is in the browser, so there is nothing to fetch here: the view reads it. */
export default function CheckoutPage() {
  return <CheckoutView />;
}
