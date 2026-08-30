import { expect, test } from "@playwright/test";

const PASSWORD = "password123";

function uniqueEmail(prefix: string): string {
  return `${prefix}-${Date.now()}@e2e.local`;
}

async function newContext(browser: import("@playwright/test").Browser) {
  const ctx = await browser.newContext();
  await ctx.addCookies([{ name: "lang", value: "en", url: "http://localhost:3000" }]);
  return ctx;
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

  await expect(page).toHaveURL("/dashboard");
  await expect(
    page.getByRole("heading", { name: new RegExp(`Welcome back, ${name.split(" ")[0]}`) }),
  ).toBeVisible();
}

test("complete journey: register → group → invite link → expense → balance → settle", async ({
  browser,
}) => {
  const ownerName = "Owner";
  const friendName = "Friend";
  const ownerEmail = uniqueEmail("owner");
  const friendEmail = uniqueEmail("friend");

  const ownerContext = await newContext(browser);
  const owner = await ownerContext.newPage();
  await register(owner, ownerName, ownerEmail);

  // Create a group with an optional logo.
  await owner.goto("/groups");
  await owner.getByRole("link", { name: "New group" }).click();
  await owner.getByLabel("Group name").fill("E2E Trip");
  await owner.locator("#logo").setInputFiles("tests/e2e/fixtures/receipt.png");
  await owner.getByRole("button", { name: "Create" }).click();
  await expect(owner).toHaveURL(/\/groups\/[0-9a-f-]+/);
  await expect(owner.getByText("Group created.")).toBeVisible();
  await expect(owner.getByRole("heading", { name: "E2E Trip" })).toBeVisible();
  await expect(owner.locator("img[src*='/logo']")).toBeVisible();

  // Create invitation link and capture it.
  const friendContext = await newContext(browser);
  const friendPage = await friendContext.newPage();

  // Click "Create invitation link" button
  await owner.getByRole("button", { name: "Create invitation link" }).click();
  // Wait for the link input to appear with the generated URL - wait for value to be non-empty
  // Use a more specific locator for the invite link input (inside the invite link card)
  const inviteLinkCard = owner.locator('section:has-text("Invitation link")').first();
  const linkInput = inviteLinkCard.locator('input[type="text"]');
  await expect(linkInput).toBeVisible({ timeout: 15000 });
  await expect.poll(async () => await linkInput.inputValue(), { timeout: 15000, intervals: [200] }).not.toBe("");
  const inviteLink = await linkInput.inputValue();
  expect(inviteLink).toContain("/join/");

  // Friend opens the invitation link directly
  await friendPage.goto(inviteLink);
  // Wait for network to be idle
  await friendPage.waitForLoadState('networkidle');
  // Debug: print full page content
  const pageContent = await friendPage.content();
  console.log('Full page content:', pageContent);

  // Friend registers via the "Register to join" link
  await friendPage.getByRole("link", { name: "Register to join" }).click();
  await expect(friendPage).toHaveURL(/\/register\?next=/);
  await friendPage.getByLabel("Name").fill(friendName);
  await friendPage.getByLabel("Email").fill(friendEmail);
  await friendPage.getByLabel("Password").fill(PASSWORD);
  await friendPage.getByRole("button", { name: "Create account" }).click();

  // After registration, should redirect to /join/{token} and auto-join
  await expect(friendPage).toHaveURL(/\/groups\/[0-9a-f-]+/);
  await expect(friendPage.getByText("You joined the group.")).toBeVisible();
  await expect(friendPage.getByRole("heading", { name: "E2E Trip" })).toBeVisible();

  // The owner's page was rendered before the friend joined, so reload to see the new member.
  await owner.reload();
  await expect(owner.getByText("2 members · IDR")).toBeVisible();

  // Owner adds an expense with a receipt image: 200000 split equally between both members.
  await owner.getByRole("heading", { name: "Add Expense" }).scrollIntoViewIfNeeded();
  await owner.getByLabel("Description").fill("Dinner");
  await owner.getByLabel("Amount").fill("200000");
  await owner.getByLabel(new RegExp(`^${friendName}$`)).check();
  await owner.locator("#receipt").setInputFiles("tests/e2e/fixtures/receipt.png");
  await owner.getByRole("button", { name: "Add Expense" }).click();
  await expect(owner.getByText("Expense added successfully.")).toBeVisible();
  await expect(owner.getByText("Dinner")).toBeVisible();
  await expect(owner.getByRole("link", { name: /View receipt for Dinner/ })).toBeVisible();

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
  await owner.goto("/dashboard");
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
  await friendContext.close();
});

test("invitation link shows preview for unauthenticated users", async ({ browser }) => {
  const ownerName = "Owner";
  const ownerEmail = uniqueEmail("owner");

  const ownerContext = await newContext(browser);
  const owner = await ownerContext.newPage();
  await register(owner, ownerName, ownerEmail);

  await owner.goto("/groups");
  await owner.getByRole("link", { name: "New group" }).click();
  await owner.getByLabel("Group name").fill("Preview Test");
  await owner.getByRole("button", { name: "Create" }).click();
  await expect(owner).toHaveURL(/\/groups\/[0-9a-f-]+/);

  // Create invitation link
  await owner.getByRole("button", { name: "Create invitation link" }).click();
  const inviteLinkCard = owner.locator('section:has-text("Invitation link")').first();
  const linkInput = inviteLinkCard.locator('input[type="text"]');
  await expect(linkInput).toBeVisible({ timeout: 15000 });
  await expect.poll(async () => await linkInput.inputValue(), { timeout: 15000, intervals: [200] }).not.toBe("");
  const inviteLink = await linkInput.inputValue();

  // New unauthenticated context
  const anonContext = await newContext(browser);
  const anonPage = await anonContext.newPage();

  // Open invitation link without auth
  await anonPage.goto(inviteLink);
  await expect(anonPage.getByRole("heading", { name: "Preview Test" })).toBeVisible();
  await expect(anonPage.getByText("Group preview")).toBeVisible();
  await expect(anonPage.getByRole("link", { name: "Sign in to join" })).toBeVisible();
  await expect(anonPage.getByRole("link", { name: "Register to join" })).toBeVisible();

  // Clicking "Sign in to join" should redirect to login with next param
  await anonPage.getByRole("link", { name: "Sign in to join" }).click();
  await expect(anonPage).toHaveURL(/\/login\?next=/);

  await ownerContext.close();
  await anonContext.close();
});

test("group page is not found for non-members", async ({ browser }) => {
  const ctx = await newContext(browser);
  const page = await ctx.newPage();
  await register(page, "Outsider", uniqueEmail("outsider"));

  await page.goto("/groups/00000000-0000-0000-0000-000000000000");
  await expect(page.getByText("Page not found")).toBeVisible();

  await ctx.close();
});
