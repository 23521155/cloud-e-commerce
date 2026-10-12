import type { Metadata } from "next";
import { CartView } from "@/components/views/CartView";
import { queryBooks } from "@/lib/catalogue";

export const metadata: Metadata = {
  title: "Basket — Marginalleya",
  description: "The copies you have set aside, ready for checkout.",
};

/** The basket is in the browser, so the server only fetches the suggestions: the newest books. */
export default async function CartPage() {
  // Stand-in for the recommender: one page of the newest books, more than are shown, so the
  // view can leave out those already in the basket
  const { books } = await queryBooks({ sort: "new" });
  return <CartView shelf={books} />;
}
