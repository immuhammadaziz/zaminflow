import type { components } from "@/lib/api/schema";

type User = components["schemas"]["User"];

export function getUserDestination(user: User): string {
  if (!user.emailVerified) return "/verify-email";
  if (user.accountStatus === "pending") return "/account/pending";
  if (user.accountStatus === "rejected") return "/account/rejected";
  if (user.accountStatus === "suspended") return "/account/suspended";
  if (user.role === "admin") return "/admin/dashboard";
  return user.role === "farmer" ? "/farmer/dashboard" : "/processor/dashboard";
}
