import type { Metadata } from "next";
import { AuthPage, type VerifyState } from "@/components/auth/Auth";
import { MotionController } from "@/components/motion/MotionController";

export const metadata: Metadata = {
  title: "Confirm your email — Marginalleya",
  // The URL carries a one-time token
  robots: { index: false },
};

const first = (v: string | string[] | undefined) => (Array.isArray(v) ? v[0] : v);

/** Where the link in the confirmation email leads: one button confirms the address. */
export default async function VerifyEmailPage({ searchParams }: PageProps<"/verify-email">) {
  const sp = await searchParams;
  const error = first(sp.error);
  const verify: VerifyState = {
    token: first(sp.token),
    done: first(sp.done) === "1",
    error: error === "invalid" || error === "failed" ? error : undefined,
  };

  return (
    <>
      <MotionController />
      <AuthPage mode="verify" verify={verify} />
    </>
  );
}
