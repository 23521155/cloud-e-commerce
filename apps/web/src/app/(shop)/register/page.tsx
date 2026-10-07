import type { Metadata } from "next";
import { AuthPage } from "@/components/auth/Auth";
import { MotionController } from "@/components/motion/MotionController";

export const metadata: Metadata = {
  title: "Create an account — Marginalleya",
};

export default function RegisterPage() {
  return (
    <>
      <MotionController />
      <AuthPage mode="register" />
    </>
  );
}
