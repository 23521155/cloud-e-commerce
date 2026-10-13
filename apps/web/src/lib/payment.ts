// Payment methods offered at checkout. VietQR and card are paid on VNPay's page; MoMo and PayPal
// are still mocked by /checkout/pay. The `id` is the value the order form submits.
export type PaymentMethodId = "vietqr" | "momo" | "cod" | "card" | "paypal";

export type PaymentMethod = {
  id: PaymentMethodId;
  name: string;
  note: string;
  icon: "qr" | "wallet" | "parcel" | "card" | "globe";
  /**
   * gateway: "Place order" hands the reader to the provider's page (VNPay, or the mock at /checkout/pay).
   * offline: nothing to pay online.
   */
  flow: "gateway" | "offline";
};

export const PAYMENT_GROUPS: { label: string; currency: string; methods: PaymentMethod[] }[] = [
  {
    label: "In Vietnam",
    currency: "VND",
    methods: [
      { id: "vietqr", name: "Bank transfer (VietQR)", note: "Scan a QR code with any banking app.", icon: "qr", flow: "gateway" },
      { id: "momo", name: "MoMo", note: "Pay in the MoMo e-wallet app.", icon: "wallet", flow: "gateway" },
      // VNPay charges in dong, so the card is paid in VND too.
      { id: "card", name: "Card", note: "Visa, Mastercard, JCB or a domestic ATM card.", icon: "card", flow: "gateway" },
      { id: "cod", name: "Cash on delivery", note: "Pay the courier when the parcel arrives.", icon: "parcel", flow: "offline" },
    ],
  },
  {
    label: "From anywhere",
    currency: "USD",
    methods: [
      { id: "paypal", name: "PayPal", note: "Pay from your PayPal account.", icon: "globe", flow: "gateway" },
    ],
  },
];

export const PAYMENT_METHODS: PaymentMethod[] = PAYMENT_GROUPS.flatMap((g) => g.methods);

export const DEFAULT_PAYMENT: PaymentMethodId = "vietqr";

export const getPaymentMethod = (id: unknown) => PAYMENT_METHODS.find((m) => m.id === id);
