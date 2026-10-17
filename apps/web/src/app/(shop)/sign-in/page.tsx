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
  if (sp.google === "cancelled") return "Google sign-in was cancelled. Choose an account to continue, or sign in below.";
  if (sp.google === "unverified") return "Google has not confirmed that account's email address, so it cannot be used here.";
  if (sp.google === "off") return "Signing in with Google is not available right now. Please use your email and password.";
  if (sp.google === "failed") return "You could not be signed in with Google. Please try again.";
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
