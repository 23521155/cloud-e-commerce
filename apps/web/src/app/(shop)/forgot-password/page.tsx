import type { Metadata } from "next";
import { AuthPage } from "@/components/auth/Auth";
import { MotionController } from "@/components/motion/MotionController";

export const metadata: Metadata = {
  title: "Reset your password — Marginalleya",
};

export default async function ForgotPasswordPage({ searchParams }: PageProps<"/forgot-password">) {
  const sp = await searchParams;
  return (
    <>
      <MotionController />
      <AuthPage mode="forgot" sent={sp.sent === "1"} />
    </>
  );
}
