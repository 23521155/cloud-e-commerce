import type { Metadata } from "next";
import { redirect } from "next/navigation";
import { SampleQr } from "@/components/checkout/PaymentDetail";
import { DotButton } from "@/components/ui/DotButton";
import { getBasket, subtotal } from "@/lib/basket";
import { getPaymentMethod } from "@/lib/payment";

export const metadata: Metadata = {
  title: "Payment — Marginalleya",
};

const price = new Intl.NumberFormat("en-US", { style: "currency", currency: "VND", maximumFractionDigits: 0 });

const first = (v: string | string[] | undefined) => (Array.isArray(v) ? v[0] : v);

/** Copy for each gateway's step. The live site redirects to the provider instead of rendering this page. */
const STEPS: Record<string, { provider: string; title: string; text: string; qr: boolean }> = {
  vietqr: { provider: "the bank transfer gateway", title: "Scan to pay", text: "Open any Vietnamese banking app, scan the code and confirm the amount.", qr: true },
  momo: { provider: "MoMo", title: "Scan with MoMo", text: "Open the MoMo app, tap Scan, and approve the payment.", qr: true },
  paypal: { provider: "PayPal", title: "Approve in PayPal", text: "On the live site this is PayPal’s own page: you log in to your PayPal account, check the amount and approve it.", qr: false },
};

export default async function PayPage({ searchParams }: PageProps<"/checkout/pay">) {
  const method = getPaymentMethod(first((await searchParams).method));
  if (!method || method.flow !== "gateway") redirect("/checkout");

  const step = STEPS[method.id];
  const total = subtotal(getBasket().filter((l) => !l.sold));

  return (
    <section className="work basket" aria-labelledby="pay-title">
      <div className="co-pay">
        <p className="co-pay__demo small-upper m-0">Demo payment page · stands in for {step.provider}</p>

        <div className="co-pay__card">
          <p className="small-upper m-0 text-oxblood-700">{method.name}</p>
          <h1 id="pay-title" className="co-pay__title">
            {step.title}
          </h1>
          <p className="co-pay__amount m-0 tabular-nums">{price.format(total)}</p>

          {step.qr && (
            <div className="co-pay__qr">
              <SampleQr seed={total + method.id.length} />
            </div>
          )}
          <p className="co-pay__text m-0">{step.text}</p>
        </div>

        <div className="co-pay__actions">
          <DotButton href={`/checkout/confirmed?method=${method.id}`} variant="solid">
            Simulate payment
          </DotButton>
          <DotButton href="/checkout">Cancel</DotButton>
        </div>
      </div>
    </section>
  );
}
