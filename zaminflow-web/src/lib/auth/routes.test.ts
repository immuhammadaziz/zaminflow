import { describe, expect, it } from "vitest";
import { getUserDestination } from "@/lib/auth/routes";
import type { components } from "@/lib/api/schema";

type User = components["schemas"]["User"];

const user: User = {
  id: "user-1",
  role: "farmer",
  accountStatus: "approved",
  email: "farmer@example.test",
  emailVerified: true,
  fullName: "Farmer User",
  createdAt: "2025-01-01T00:00:00Z",
};

describe("getUserDestination", () => {
  it("routes unverified users to email verification before account status", () => {
    expect(
      getUserDestination({
        ...user,
        emailVerified: false,
        accountStatus: "pending",
      }),
    ).toBe("/verify-email");
  });

  it.each([
    ["pending", "/account/pending"],
    ["rejected", "/account/rejected"],
    ["suspended", "/account/suspended"],
  ] as const)(
    "routes %s accounts to their status page",
    (accountStatus, path) => {
      expect(getUserDestination({ ...user, accountStatus })).toBe(path);
    },
  );

  it("routes approved accounts to their role dashboard", () => {
    expect(getUserDestination(user)).toBe("/farmer/dashboard");
    expect(getUserDestination({ ...user, role: "processor" })).toBe(
      "/processor/dashboard",
    );
    expect(getUserDestination({ ...user, role: "admin" })).toBe(
      "/admin/dashboard",
    );
  });
});
