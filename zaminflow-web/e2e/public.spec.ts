import AxeBuilder from "@axe-core/playwright";
import { expect, test } from "@playwright/test";

test.beforeEach(async ({ page }) => {
  await page.route("**/api/v1/auth/refresh", (route) =>
    route.fulfill({ status: 401, body: "" }),
  );
});

test("localized landing page provides clear sign-in and signup actions", async ({
  page,
}) => {
  await page.goto("/en");
  await expect(
    page.getByRole("heading", { name: /from farm supply.*to trusted trade/i }),
  ).toBeVisible();
  await expect(
    page.getByRole("link", { name: "Sign in" }).first(),
  ).toHaveAttribute("href", "/en/sign-in");
  await expect(
    page.getByRole("link", { name: "Join ZaminFlow" }).first(),
  ).toHaveAttribute("href", "/en/sign-up");
});

test("sign-in form exposes accessible required-field validation", async ({
  page,
}) => {
  await page.goto("/en/sign-in");
  await page.getByRole("button", { name: "Sign in" }).click();
  await expect(page.getByLabel("Email address")).toHaveAttribute(
    "aria-invalid",
    "true",
  );
  await expect(page.getByRole("textbox", { name: "Password" })).toHaveAttribute(
    "aria-invalid",
    "true",
  );
});

test("page uses a nonce-based content security policy", async ({ page }) => {
  const response = await page.goto("/en");
  const policy = response?.headers()["content-security-policy"];
  expect(policy).toMatch(/script-src 'self' 'nonce-[^']+' 'strict-dynamic'/);
  expect(response?.headers()["x-content-type-options"]).toBe("nosniff");
  expect(response?.headers()["x-frame-options"]).toBe("DENY");
});

test("unknown localized route renders the branded not-found page", async ({
  page,
}) => {
  const response = await page.goto("/en/does-not-exist");
  expect(response?.status()).toBe(404);
  await expect(
    page.getByRole("heading", { name: "This page could not be found." }),
  ).toBeVisible();
});

test("landing page has no automated accessibility violations", async ({
  page,
}) => {
  await page.goto("/en");
  const results = await new AxeBuilder({ page }).analyze();
  expect(results.violations).toEqual([]);
});
