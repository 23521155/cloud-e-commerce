import type { Metadata } from "next";
import { redirect } from "next/navigation";
import { Account, isAccountError, type Profile, type SavedNotice } from "@/components/account/Account";
import { MotionController } from "@/components/motion/MotionController";
import { getSession } from "@/lib/session";
import { me } from "@/lib/user-api";

export const metadata: Metadata = {
  title: "Your account — Marginalleya",
  description: "Update your personal details, delivery address and password.",
};

export default async function AccountPage({ searchParams }: PageProps<"/account">) {
  const sp = await searchParams;
  const saved = (["profile", "password", "verify-sent", "mfa-on"] as const).find((s) => s === sp.saved) as SavedNotice | undefined;
  const error = isAccountError(sp.error) ? sp.error : undefined;

  const session = await getSession();
  if (!session) redirect("/sign-in?next=/account");

  const result = await me(session.token);
  if (!result.ok && result.reason === "signed-out") redirect("/sign-in?next=/account");
  if (!result.ok) throw new Error("Your account could not be loaded right now.");

  const { account } = result;
  const profile: Profile = {
    name: account.name,
    email: account.email,
    phone: account.phone,
    address: account.address,
    city: account.city,
    postal: account.postalCode,
    country: account.country || "Vietnam",
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

        <Account profile={profile} emailVerified={account.emailVerifiedAt !== null} twoStepSince={account.totpEnabledAt} saved={saved} error={error} />
      </section>
    </>
  );
}
