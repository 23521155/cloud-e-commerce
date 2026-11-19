import type { Metadata } from "next";
import { AuthPage, type ResetState } from "@/components/auth/Auth";
import { MotionController } from "@/components/motion/MotionController";

export const metadata: Metadata = {
  title: "Choose a new password — Marginalleya",
  // The URL carries a one-time token
  robots: { index: false },
};

const ERRORS = ["invalid-link", "weak", "mismatch", "failed"] as const;

const first = (v: string | string[] | undefined) => (Array.isArray(v) ? v[0] : v);

/** Where the link in the reset email leads. */
export default async function ResetPasswordPage({ searchParams }: PageProps<"/reset-password">) {
  const sp = await searchParams;
  const error = ERRORS.find((e) => e === first(sp.error));
  const reset: ResetState = { token: first(sp.token), error };

  return (
    <>
      <MotionController />
      <AuthPage mode="reset" reset={reset} />
    </>
  );
}
