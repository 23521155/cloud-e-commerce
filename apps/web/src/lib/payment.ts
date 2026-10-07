// Payment methods offered at checkout. Mock only: no gateway is wired yet.
// The `id` is the value the order form submits; map it to a gateway when one is integrated.
export type PaymentMethodId = "vietqr" | "momo" | "cod" | "card" | "paypal";

export type PaymentMethod = {
  id: PaymentMethodId;
  name: string;
  note: string;
  icon: "qr" | "wallet" | "parcel" | "card" | "globe";
  /**
   * gateway: "Place order" hands the reader to the provider's page (mocked by /checkout/pay).
   * onsite: details are entered on the checkout page itself (the provider's hosted fields).
   * offline: nothing to pay online.
   */
  flow: "gateway" | "onsite" | "offline";
};

export const PAYMENT_GROUPS: { label: string; currency: string; methods: PaymentMethod[] }[] = [
  {
    label: "In Vietnam",
    currency: "VND",
    methods: [
      { id: "vietqr", name: "Bank transfer (VietQR)", note: "Scan a QR code with any banking app.", icon: "qr", flow: "gateway" },
      { id: "momo", name: "MoMo", note: "Pay in the MoMo e-wallet app.", icon: "wallet", flow: "gateway" },
      { id: "cod", name: "Cash on delivery", note: "Pay the courier when the parcel arrives.", icon: "parcel", flow: "offline" },
    ],
  },
  {
    label: "From anywhere",
    currency: "USD",
    methods: [
      { id: "card", name: "Card", note: "Visa, Mastercard or JCB.", icon: "card", flow: "onsite" },
      { id: "paypal", name: "PayPal", note: "Pay from your PayPal account.", icon: "globe", flow: "gateway" },
    ],
  },
];

export const PAYMENT_METHODS: PaymentMethod[] = PAYMENT_GROUPS.flatMap((g) => g.methods);

export const DEFAULT_PAYMENT: PaymentMethodId = "vietqr";

export const getPaymentMethod = (id: unknown) => PAYMENT_METHODS.find((m) => m.id === id);
