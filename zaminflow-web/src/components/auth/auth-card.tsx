import type { ReactNode } from "react";
import { Link } from "@/i18n/navigation";

export function AuthCard({
  title,
  description,
  children,
}: Readonly<{
  title: string;
  description: string;
  children: ReactNode;
}>) {
  return (
    <main
      id="main-content"
      className="mx-auto flex min-h-screen max-w-7xl items-center justify-center px-5 py-12 sm:px-8"
    >
      <section className="rounded-panel border-border bg-surface w-full max-w-lg border p-6 shadow-sm sm:p-10">
        <Link
          href="/"
          className="text-sage inline-flex items-center gap-2 text-sm font-bold"
        >
          <span aria-hidden="true">←</span> ZaminFlow
        </Link>
        <h1 className="font-display text-ink mt-8 text-3xl font-bold tracking-tight">
          {title}
        </h1>
        <p className="text-muted mt-3 text-sm leading-6">{description}</p>
        <div className="mt-8">{children}</div>
      </section>
    </main>
  );
}
