import { readFile } from "node:fs/promises";

const locales = ["en", "ru", "uz"];
const backendErrorCodes = JSON.parse(
  await readFile(
    new URL("../contracts/error-codes.json", import.meta.url),
    "utf8",
  ),
);
const messages = Object.fromEntries(
  await Promise.all(
    locales.map(async (locale) => [
      locale,
      JSON.parse(
        await readFile(
          new URL(`../src/messages/${locale}.json`, import.meta.url),
          "utf8",
        ),
      ),
    ]),
  ),
);

function flattenKeys(value, prefix = "") {
  return Object.entries(value).flatMap(([key, child]) => {
    const path = prefix ? `${prefix}.${key}` : key;
    return child && typeof child === "object" && !Array.isArray(child)
      ? flattenKeys(child, path)
      : [path];
  });
}

const baseline = flattenKeys(messages.en).sort();
let hasErrors = false;
for (const locale of locales.slice(1)) {
  const keys = flattenKeys(messages[locale]).sort();
  const missing = baseline.filter((key) => !keys.includes(key));
  const extra = keys.filter((key) => !baseline.includes(key));
  if (missing.length || extra.length) {
    hasErrors = true;
    console.error(
      `${locale}: missing [${missing.join(", ")}], extra [${extra.join(", ")}]`,
    );
  }
}

for (const locale of locales) {
  for (const code of backendErrorCodes) {
    if (typeof messages[locale].errors?.[code] !== "string") {
      hasErrors = true;
      console.error(`${locale}: missing backend error translation ${code}`);
    }
  }
}

if (hasErrors) process.exitCode = 1;
else
  console.log(
    `All ${locales.length} locales have matching keys and error translations.`,
  );
