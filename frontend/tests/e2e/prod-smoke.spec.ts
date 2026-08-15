import { expect, test } from "@playwright/test";

const PASSWORD = "password123";
const suffix = Date.now();

test("production smoke: register → login → group → expense → balance → delete", async ({
  browser,
}) => {
  const ctx = await browser.newContext();
  const page = await ctx.newPage();

  const name = `Prod ${suffix}`;
  const email = `prod-smoke-${suffix}@e2e.local`;

  await page.goto("/register");
  await page.getByLabel("Name").fill(name);
  await page.getByLabel("Email").fill(email);
  await page.getByLabel("Password").fill(PASSWORD);
  await page.getByRole("button", { name: "Create account" }).click();
  await expect(page).toHaveURL(/\/login/);
  await page.getByLabel("Email").fill(email);
  await page.getByLabel("Password").fill(PASSWORD);
  await page.getByRole("button", { name: "Sign in" }).click();
  await expect(page).toHaveURL("/");

  await page.goto("/groups");
  await page.getByRole("link", { name: "New group" }).click();
  await page.getByLabel("Group name").fill(`Prod Group ${suffix}`);
  await page.getByRole("button", { name: "Create group" }).click();
  await expect(page).toHaveURL(/\/groups\/[0-9a-f-]+/);
  await expect(page.getByText("Group created.")).toBeVisible();

  await page.getByRole("heading", { name: "Add Expense" }).scrollIntoViewIfNeeded();
  await page.getByLabel("Description").fill("Lunch");
  await page.getByLabel("Amount").fill("150000");
  await page.getByRole("button", { name: "Add Expense" }).click();
  await expect(page.getByText("Expense added successfully.")).toBeVisible();
  await expect(page.getByText("Lunch")).toBeVisible();

  await page.goto("/");
  await expect(page.getByText("Prod Group", { exact: false }).first()).toBeVisible();
  await expect(page.getByText("Rp150.000").first()).toBeVisible();

  await page.goto("/groups");
  await page.getByText(`Prod Group ${suffix}`, { exact: true }).first().click();
  await page.getByRole("button", { name: "Delete group" }).click();
  await page.getByRole("dialog").getByRole("button", { name: "Delete group" }).click();
  await expect(page).toHaveURL(/\/groups\?success=group-deleted/);

  await ctx.close();
});