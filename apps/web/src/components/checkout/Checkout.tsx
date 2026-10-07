import Link from "next/link";
import { Seal } from "@/components/cart/Basket";
import { PlaceOrderButton, Stepper } from "@/components/checkout/CheckoutClient";
import { PaymentDetail } from "@/components/checkout/PaymentDetail";
import type { BasketLine } from "@/lib/basket";
import { DEFAULT_PAYMENT, PAYMENT_GROUPS, type PaymentMethod } from "@/lib/payment";

const price = new Intl.NumberFormat("en-US", { style: "currency", currency: "VND", maximumFractionDigits: 0 });

const COUNTRIES = ["Vietnam", "United States", "United Kingdom", "Australia", "Canada", "Japan", "Singapore", "Other country"];

type CheckoutProps = { lines: BasketLine[]; total: number; action: (formData: FormData) => Promise<void> };

function Field({ label, name, hint, className = "", ...input }: { label: string; name: string; hint?: string; className?: string } & React.InputHTMLAttributes<HTMLInputElement>) {
  return (
    <label className={`co__field ${className}`}>
      <span className="co__label small-upper">{label}</span>
      <input name={name} className="co__input" {...input} />
      {hint && <span className="co__hint">{hint}</span>}
    </label>
  );
}

/** A section of the form drawn as a card-index card, like the catalogue search. */
function Card({ title, note, children }: { title: string; note?: string; children: React.ReactNode }) {
  return (
    <fieldset className="co__card">
      <legend className="sr-only">{title}</legend>
      <div className="catalogue__card-head small-upper" aria-hidden="true">
        <span>{title}</span>
        {note && <span>{note}</span>}
      </div>
      <div className="co__grid">{children}</div>
    </fieldset>
  );
}

function ContactCard() {
  return (
    <Card title="Contact" note="For your receipt">
      <Field label="Email" name="email" type="email" autoComplete="email" required className="co__full" placeholder="you@example.com" />
      <Field label="Phone" name="phone" type="tel" autoComplete="tel" required className="co__full" hint="Only used by the courier." />
    </Card>
  );
}

function DeliveryCard() {
  return (
    <Card title="Delivery" note="Shipping cost confirmed later">
      <Field label="Full name" name="name" autoComplete="name" required className="co__full" />
      <Field label="Address" name="address" autoComplete="street-address" required className="co__full" />
      <Field label="City" name="city" autoComplete="address-level2" required />
      <Field label="Postal code" name="postal" autoComplete="postal-code" />
      <label className="co__field co__full">
        <span className="co__label small-upper">Country</span>
        <select name="country" className="co__input co__select" autoComplete="country-name" defaultValue="Vietnam" required>
          {COUNTRIES.map((c) => (
            <option key={c}>{c}</option>
          ))}
        </select>
      </label>
    </Card>
  );
}

/** Line icons for the payment methods, one stroke weight. Not brand logos. */
function PayIcon({ icon }: { icon: PaymentMethod["icon"] }) {
  const paths: Record<PaymentMethod["icon"], React.ReactNode> = {
    qr: (
      <>
        <rect x="3" y="3" width="7" height="7" rx="1" />
        <rect x="14" y="3" width="7" height="7" rx="1" />
        <rect x="3" y="14" width="7" height="7" rx="1" />
        <path d="M14 14h3v3h-3zM20 14v1M14 20h1M18 18h3v3h-3" />
      </>
    ),
    wallet: (
      <>
        <path d="M4 7h14a2 2 0 0 1 2 2v9a2 2 0 0 1-2 2H5a1 1 0 0 1-1-1V7z" />
        <path d="M4 7l11-3v3M16 13.5h4" />
      </>
    ),
    parcel: (
      <>
        <path d="M3 7.5L12 3l9 4.5v9L12 21l-9-4.5z" />
        <path d="M3 7.5L12 12l9-4.5M12 12v9M7.5 5.25l9 4.5" />
      </>
    ),
    card: (
      <>
        <rect x="3" y="5.5" width="18" height="13" rx="2" />
        <path d="M3 10h18M7 15h4" />
      </>
    ),
    globe: (
      <>
        <circle cx="12" cy="12" r="9" />
        <path d="M3 12h18M12 3c3 3.2 3 14.8 0 18M12 3c-3 3.2-3 14.8 0 18" />
      </>
    ),
  };
  return (
    <svg className="co__option-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      {paths[icon]}
    </svg>
  );
}

function PaymentCard({ total }: { total: number }) {
  return (
    <Card title="Payment" note="Demo: nothing is charged">
      {PAYMENT_GROUPS.map((group) => (
        <div key={group.label} className="co__full co__pay-group" role="radiogroup" aria-label={`Payment methods ${group.label.toLowerCase()}`}>
          <p className="co__pay-label small-upper m-0">
            {group.label} <span>· {group.currency}</span>
          </p>
          <div className="co__pay">
            {group.methods.map((m) => (
              <label key={m.id} className="co__option">
                <input type="radio" name="payment" value={m.id} defaultChecked={m.id === DEFAULT_PAYMENT} required />
                <PayIcon icon={m.icon} />
                <span className="co__option-title">{m.name}</span>
                <span className="co__option-note">{m.note}</span>
              </label>
            ))}
          </div>
        </div>
      ))}
      {/* CSS shows the panel whose method is checked */}
      <div className="co__full co__details" aria-live="polite">
        {PAYMENT_GROUPS.flatMap((g) => g.methods).map((m) => (
          <div key={m.id} className="co__detail" data-for={m.id}>
            <PaymentDetail method={m.id} total={total} />
          </div>
        ))}
      </div>
    </Card>
  );
}

/** The basket card, folded small: what is being bought and for how much. */
function OrderCard({ lines, total }: { lines: BasketLine[]; total: number }) {
  return (
    <section className="lib-card co__order" aria-labelledby="order-title">
      <div className="lib-card__frame">
        <h2 id="order-title" className="lib-card__head">
          Order card
        </h2>
        <div className="lib-card__top">
          <dl className="lib-card__fields">
            <div>
              <dt>Copies</dt>
              <dd>{lines.length}</dd>
            </div>
            <div>
              <dt>Shipping</dt>
              <dd>To be confirmed</dd>
            </div>
          </dl>
          <Seal />
        </div>
        <table className="lib-card__table">
          <caption className="sr-only">Copies in this order</caption>
          <thead>
            <tr>
              <th scope="col">No.</th>
              <th scope="col">Copy</th>
              <th scope="col" className="text-right">
                Price
              </th>
            </tr>
          </thead>
          <tbody>
            {lines.map(({ book }, i) => (
              <tr key={book.slug}>
                <td className="lib-card__no">{i + 1}.</td>
                <td>
                  <span className="lib-card__title">{book.title}</span>
                  <span className="lib-card__by">
                    {book.author}, {book.year}
                  </span>
                </td>
                <td className="lib-card__price">
                  <span className="lib-card__amount">{price.format(book.price)}</span>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
        <div className="lib-card__foot">
          <span className="lib-card__marks" aria-hidden="true">
            ❦ ❦ ❦
          </span>
          <p className="lib-card__subtotal m-0">
            <span className="small-upper">Total</span>
            <span className="lib-card__total">{price.format(total)}</span>
          </p>
        </div>
      </div>
      <Link href="/cart" className="co__edit small-upper under-hover">
        Edit basket
      </Link>
    </section>
  );
}

const Terms = () => <p className="co__terms m-0">Shipping is not included in this total yet.</p>;

/** One step at a time: contact, delivery, then payment with the order card. */
export function Checkout({ lines, total, action }: CheckoutProps) {
  return (
    <form action={action} className="co co--steps">
      <Stepper
        steps={[
          { key: "contact", label: "Contact", content: <ContactCard /> },
          { key: "delivery", label: "Delivery", content: <DeliveryCard /> },
          {
            key: "review",
            label: "Payment & review",
            content: (
              <>
                <PaymentCard total={total} />
                <OrderCard lines={lines} total={total} />
                <Terms />
              </>
            ),
          },
        ]}
        last={<PlaceOrderButton />}
      />
    </form>
  );
}

