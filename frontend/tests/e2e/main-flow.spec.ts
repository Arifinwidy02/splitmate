import { expect, test } from "@playwright/test";

const PASSWORD = "password123";

function uniqueEmail(prefix: string): string {
  return `${prefix}-${Date.now()}@e2e.local`;
}

async function register(page: import("@playwright/test").Page, name: string, email: string) {
  await page.goto("/register");
  await page.getByLabel("Name").fill(name);
  await page.getByLabel("Email").fill(email);
  await page.getByLabel("Password").fill(PASSWORD);
  await page.getByRole("button", { name: "Create account" }).click();

  // Registration redirects to the sign-in page (part of the official flow).
  await expect(page).toHaveURL("/login");
  await page.getByLabel("Email").fill(email);
  await page.getByLabel("Password").fill(PASSWORD);
  await page.getByRole("button", { name: "Sign in" }).click();

  await expect(page).toHaveURL("/");
  await expect(
    page.getByRole("heading", { name: new RegExp(`Welcome back, ${name.split(" ")[0]}`) }),
  ).toBeVisible();
}

test("complete journey: register → group → invite → expense → balance → settle", async ({
  browser,
}) => {
  const ownerName = "Owner";
  const friendName = "Friend";
  const ownerEmail = uniqueEmail("owner");
  const friendEmail = uniqueEmail("friend");

  const ownerContext = await browser.newContext();
  const owner = await ownerContext.newPage();
  await register(owner, ownerName, ownerEmail);

  // Create a group.
  await owner.goto("/groups");
  await owner.getByRole("link", { name: "New group" }).click();
  await owner.getByLabel("Group name").fill("E2E Trip");
  await owner.getByRole("button", { name: "Create group" }).click();
  await expect(owner).toHaveURL(/\/groups\/[0-9a-f-]+/);
  await expect(owner.getByText("Group created.")).toBeVisible();
  await expect(owner.getByRole("heading", { name: "E2E Trip" })).toBeVisible();

  // Invite the friend and capture the token from the UI.
  const friendPage = await browser.newPage();
  const friendEmailRaw = friendEmail;
  await owner.getByLabel("Email").fill(friendEmailRaw);
  await owner.getByRole("button", { name: "Invite" }).click();
  const token = await owner
    .locator("code")
    .filter({ hasText: /^[A-Za-z0-9_-]{20,}$/ })
    .textContent();
  expect(token).toBeTruthy();

  // Friend registers and joins with the token.
  await register(friendPage, friendName, friendEmail);
  await friendPage.goto("/groups");
  await friendPage.getByRole("heading", { name: "Join a group" }).scrollIntoViewIfNeeded();
  await friendPage.getByLabel("Invitation token").fill(token ?? "");
  await friendPage.getByRole("button", { name: "Join group" }).click();
  await expect(friendPage).toHaveURL(/\/groups\/[0-9a-f-]+/);
  await expect(friendPage.getByText("You joined the group.")).toBeVisible();
  await expect(friendPage.getByRole("heading", { name: "E2E Trip" })).toBeVisible();

  // Owner adds an expense: 200000 split equally between both members.
  await owner.getByRole("heading", { name: "Add Expense" }).scrollIntoViewIfNeeded();
  await owner.getByLabel("Description").fill("Dinner");
  await owner.getByLabel("Amount").fill("200000");
  await owner.getByLabel(new RegExp(`^${friendName}$`)).check();
  await owner.getByRole("button", { name: "Add Expense" }).click();
  await expect(owner.getByText("Expense added successfully.")).toBeVisible();
  await expect(owner.getByText("Dinner")).toBeVisible();

  // Balances: owner paid, friend owes half.
  await expect(owner.getByText("+Rp100.000").first()).toBeVisible();
  await friendPage.reload();
  await expect(friendPage.getByText("-Rp100.000").first()).toBeVisible();

  // Friend settles via quick settle.
  await expect(friendPage.getByText(`You owe ${ownerName}`)).toBeVisible();
  await friendPage.getByRole("button", { name: "Record payment" }).first().click();
  await expect(friendPage.getByText("Payment recorded.")).toBeVisible();

  // Both balances settled.
  await expect(friendPage.getByText("Settled").first()).toBeVisible();
  await owner.reload();
  await expect(owner.getByText("Settled").first()).toBeVisible();

  // Dashboard reflects the expense and settlement totals.
  await owner.goto("/");
  await expect(owner.getByText("E2E Trip", { exact: true }).first()).toBeVisible();
  await expect(owner.getByText("Rp200.000").first()).toBeVisible();

  // Cleanup: the admin deletes the group so test data does not accumulate.
  await owner.goto("/groups");
  await owner.getByText("E2E Trip", { exact: true }).first().click();
  await owner.getByRole("button", { name: "Delete group" }).click();
  await owner.getByRole("dialog").getByRole("button", { name: "Delete group" }).click();
  await expect(owner).toHaveURL(/\/groups\?success=group-deleted/);
  await expect(owner.getByText("Group deleted.")).toBeVisible();

  await ownerContext.close();
  await friendPage.context().close();
});

test("group page is not found for non-members", async ({ browser }) => {
  const ctx = await browser.newContext();
  const page = await ctx.newPage();
  await register(page, "Outsider", uniqueEmail("outsider"));

  await page.goto("/groups/00000000-0000-0000-0000-000000000000");
  await expect(page.getByText("Page not found")).toBeVisible();

  await ctx.close();
});
