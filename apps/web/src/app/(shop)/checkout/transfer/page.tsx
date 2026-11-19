import type { Metadata } from "next";
import { redirect } from "next/navigation";
import { TransferView } from "@/components/views/TransferView";
import { readPendingTransfer } from "@/lib/transfer";

export const metadata: Metadata = {
  title: "Pay by bank transfer — Marginalleya",
};

/**
 * Where a VietQR order is paid: the QR code and the bank details, until the transfer arrives. The
 * order comes from the pending-transfer cookie set when the order was placed. The email in it stays
 * on the server: only what is shown goes to the page.
 */
export default async function TransferPage() {
  const pending = await readPendingTransfer();
  if (!pending) redirect("/checkout");

  return <TransferView code={pending.code} expiresAt={pending.expiresAt} transfer={pending.transfer} />;
}
