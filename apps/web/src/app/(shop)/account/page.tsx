import type { Metadata } from "next";
import { redirect } from "next/navigation";
import { Account, type Profile, type SavedNotice } from "@/components/account/Account";
import { MotionController } from "@/components/motion/MotionController";
import { getSession } from "@/lib/session";
import { me } from "@/lib/user-api";

export const metadata: Metadata = {
  title: "Your account — Marginalleya",
  description: "Update your personal details, delivery address and password.",
};

export default async function AccountPage({ searchParams }: PageProps<"/account">) {
  const sp = await searchParams;
  const saved: SavedNotice | undefined = sp.saved === "profile" || sp.saved === "password" ? sp.saved : undefined;

  const session = await getSession();
  if (!session) redirect("/sign-in?next=/account");

  const result = await me(session.token);
  if (!result.ok && result.reason === "signed-out") redirect("/sign-in?next=/account");
  if (!result.ok) throw new Error("Your account could not be loaded right now.");

  // The delivery address is not kept by the user service yet.
  const profile: Profile = {
    name: result.account.name,
    email: result.account.email,
    phone: "",
    address: "",
    city: "",
    postal: "",
    country: "Vietnam",
  };

  return (
    <>
      <MotionController />

      <section className="work basket" aria-labelledby="account-title">
        <div className="catalogue__head">
          <h1 id="account-title" data-reveal className="big-sans m-0 text-[clamp(2.4rem,6vw,7rem)]">
            <span className="line">
              <span>Your</span>
            </span>
            <span className="line">
              <span className="em-italic text-gold-400">account</span>
            </span>
          </h1>
          <p className="m-0 max-w-[34ch] text-[clamp(17px,1.3vw,21px)] leading-relaxed text-parchment-200">
            Keep your details and delivery address up to date, and change your password.
          </p>
        </div>

        <Account profile={profile} saved={saved} />
      </section>
    </>
  );
}
