import type { Metadata } from "next";
import { redirect } from "next/navigation";
import { PayView, type PayStep } from "@/components/views/PayView";
import { getPaymentMethod } from "@/lib/payment";

export const metadata: Metadata = {
  title: "Payment — Marginalleya",
};

const first = (v: string | string[] | undefined) => (Array.isArray(v) ? v[0] : v);

/** Copy for each gateway's step. The live site redirects to the provider instead of rendering this page. */
const STEPS: Record<string, PayStep> = {
  vietqr: { provider: "the bank transfer gateway", title: "Scan to pay", text: "Open any Vietnamese banking app, scan the code and confirm the amount.", qr: true },
  momo: { provider: "MoMo", title: "Scan with MoMo", text: "Open the MoMo app, tap Scan, and approve the payment.", qr: true },
  paypal: { provider: "PayPal", title: "Approve in PayPal", text: "On the live site this is PayPal’s own page: you log in to your PayPal account, check the amount and approve it.", qr: false },
};

export default async function PayPage({ searchParams }: PageProps<"/checkout/pay">) {
  const method = getPaymentMethod(first((await searchParams).method));
  if (!method || method.flow !== "gateway") redirect("/checkout");

  return <PayView method={method} step={STEPS[method.id]} />;
}
