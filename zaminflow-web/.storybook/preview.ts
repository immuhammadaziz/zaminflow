import type { Preview } from "@storybook/nextjs-vite";
import { createElement } from "react";
import { NextIntlClientProvider } from "next-intl";
import messages from "../src/messages/en.json";
import "../src/app/globals.css";

const preview: Preview = {
  decorators: [
    (Story) =>
      createElement(
        NextIntlClientProvider,
        { locale: "en", messages },
        createElement(Story),
      ),
  ],
  parameters: {
    actions: { argTypesRegex: "^on[A-Z].*" },
    controls: {
      matchers: {
        color: /(background|color)$/i,
        date: /Date$/i,
      },
    },
    a11y: {
      test: "error",
    },
  },
};

export default preview;
