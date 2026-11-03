import Link from "next/link";

export default function NotFound() {
  return (
    <main id="main-content" className="mx-auto max-w-3xl px-5 py-24 sm:px-8">
      <p className="text-sage text-sm font-semibold">ZaminFlow</p>
      <h1 className="font-display text-ink mt-4 text-4xl font-bold">
        Page not found
      </h1>
      <p className="text-muted mt-4">
        The page may have moved or the address may be incorrect.
      </p>
      <Link
        href="/uz"
        className="bg-sage mt-8 inline-flex min-h-11 items-center rounded-full px-5 font-semibold text-white"
      >
        Return to ZaminFlow
      </Link>
    </main>
  );
}
