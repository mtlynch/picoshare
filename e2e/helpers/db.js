export async function wipeDB(page) {
  await page.goto("/api/debug/db/wipe");
}
