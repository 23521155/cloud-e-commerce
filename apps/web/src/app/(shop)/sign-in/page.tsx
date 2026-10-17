import type { Metadata } from "next";
import { AuthPage } from "@/components/auth/Auth";
import { MotionController } from "@/components/motion/MotionController";

export const metadata: Metadata = {
  title: "Sign in — Marginalleya",
};

/** The line above the form after registering or after a reset. */
function notice(sp: Record<string, string | string[] | undefined>): string | undefined {
  if (sp.reset === "1") return "Your new password is set. Sign in with it.";
  if (sp.mfa === "expired") return "That sign-in took too long or had too many wrong codes. Please sign in again.";
  if (sp.registered === "1") return "Your account is ready. Sign in to continue.";
  return undefined;
}

export default async function SignInPage({ searchParams }: PageProps<"/sign-in">) {
  const sp = await searchParams;
  return (
    <>
      <MotionController />
      <AuthPage mode="sign-in" notice={notice(sp)} next={typeof sp.next === "string" ? sp.next : undefined} />
    </>
  );
}
