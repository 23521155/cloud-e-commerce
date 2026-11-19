import type { Metadata } from "next";
import { redirect } from "next/navigation";
import { AuthPage } from "@/components/auth/Auth";
import { MotionController } from "@/components/motion/MotionController";
import { codeStepToken } from "@/lib/session";

export const metadata: Metadata = {
  title: "Your code — Marginalleya",
};

/** The second step of signing in, for accounts with two-step sign-in on. */
export default async function SignInCodePage({ searchParams }: PageProps<"/sign-in/code">) {
  // Only reachable after the password: the first step leaves its token in a cookie.
  if (!(await codeStepToken())) redirect("/sign-in");

  const sp = await searchParams;
  return (
    <>
      <MotionController />
      <AuthPage mode="code" next={typeof sp.next === "string" ? sp.next : undefined} />
    </>
  );
}
