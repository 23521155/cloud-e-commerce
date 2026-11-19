// Payment methods offered at checkout: bank transfer (VietQR) and cash on delivery. The union keeps
// the ids the demo data and the order service still know. The `id` is the value the order form submits.
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
      { id: "cod", name: "Cash on delivery", note: "Pay the courier when the parcel arrives.", icon: "parcel", flow: "offline" },
    ],
  },
];

export const PAYMENT_METHODS: PaymentMethod[] = PAYMENT_GROUPS.flatMap((g) => g.methods);

export const DEFAULT_PAYMENT: PaymentMethodId = "vietqr";

export const getPaymentMethod = (id: unknown) => PAYMENT_METHODS.find((m) => m.id === id);
