import type { Metadata } from "next";
import { AuthPage } from "@/components/auth/Auth";
import { MotionController } from "@/components/motion/MotionController";

export const metadata: Metadata = {
  title: "Sign in — Marginalleya",
};

export default function SignInPage() {
  return (
    <>
      <MotionController />
      <AuthPage mode="sign-in" />
    </>
  );
}
