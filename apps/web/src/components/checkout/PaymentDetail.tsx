import { CardFields } from "@/components/checkout/CheckoutClient";
import type { PaymentMethodId } from "@/lib/payment";

const price = new Intl.NumberFormat("en-US", { style: "currency", currency: "VND", maximumFractionDigits: 0 });

/**
 * A decorative QR-like grid (finder squares + seeded modules). Not a real code: it stands in until
 * the gateway returns the actual image.
 */
export function SampleQr({ seed }: { seed: number }) {
  const size = 25;
  let s = seed || 1;
  const rand = () => (s = (s * 1103515245 + 12345) & 0x7fffffff) / 0x7fffffff;
  const inFinder = (x: number, y: number) =>
    [
      [0, 0],
      [size - 7, 0],
      [0, size - 7],
    ].some(([fx, fy]) => x >= fx - 1 && x <= fx + 7 && y >= fy - 1 && y <= fy + 7);
  const cells: string[] = [];
  for (let y = 0; y < size; y++) for (let x = 0; x < size; x++) if (!inFinder(x, y) && rand() > 0.52) cells.push(`M${x} ${y}h1v1h-1z`);
  const finder = (x: number, y: number) => `M${x} ${y}h7v7h-7zM${x + 1} ${y + 1}v5h5v-5zM${x + 2} ${y + 2}h3v3h-3z`;

  return (
    <figure className="co-qr m-0">
      <svg viewBox={`-2 -2 ${size + 4} ${size + 4}`} role="img" aria-label="Sample QR code (demo, not scannable)">
        <rect x="-2" y="-2" width={size + 4} height={size + 4} fill="#fcf6ea" />
        <path d={[finder(0, 0), finder(size - 7, 0), finder(0, size - 7), ...cells].join("")} fill="#1c0d07" fillRule="evenodd" />
      </svg>
      <figcaption className="small-upper">Sample · not scannable</figcaption>
    </figure>
  );
}

/** What happens with the selected method, shown under the payment choices at checkout. */
export function PaymentDetail({ method, total }: { method: PaymentMethodId; total: number }) {
  const amount = <strong className="co-detail__amount">{price.format(total)}</strong>;

  switch (method) {
    case "vietqr":
      return (
        <div className="co-detail">
          <p className="co-detail__title">Pay by bank transfer</p>
          <p className="co-detail__text">After you place the order, the payment page opens with a VietQR code for {amount}. Scan it with any Vietnamese banking app to pay.</p>
        </div>
      );
    case "momo":
      return (
        <div className="co-detail">
          <p className="co-detail__title">Pay with MoMo</p>
          <p className="co-detail__text">After you place the order, MoMo’s payment page opens. Scan its code with the MoMo app, or open the app on your phone, and approve {amount}.</p>
        </div>
      );
    case "cod":
      return (
        <div className="co-detail">
          <p className="co-detail__title">Cash on delivery</p>
          <p className="co-detail__text">Pay the courier {amount} plus shipping when the parcel arrives. Having the exact amount ready helps.</p>
        </div>
      );
    case "card":
      return (
        <div className="co-detail">
          <p className="co-detail__title">Card details · {amount}</p>
          <CardFields />
        </div>
      );
    case "paypal":
      return (
        <div className="co-detail">
          <p className="co-detail__title">Pay with PayPal</p>
          <p className="co-detail__text">After you place the order, PayPal opens. Log in to your PayPal account, approve {amount}, and you come straight back here.</p>
        </div>
      );
  }
}
