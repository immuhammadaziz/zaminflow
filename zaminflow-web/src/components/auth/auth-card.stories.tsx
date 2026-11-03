import type { Meta, StoryObj } from "@storybook/nextjs-vite";
import { AuthCard } from "@/components/auth/auth-card";

const meta = {
  title: "Auth/AuthCard",
  component: AuthCard,
  args: {
    title: "Welcome back",
    description: "Sign in to your ZaminFlow account.",
    children: (
      <div className="space-y-4">
        <label className="block text-sm font-semibold">
          Email address
          <input
            type="email"
            className="border-border mt-2 min-h-12 w-full rounded-lg border bg-white px-4"
          />
        </label>
        <button
          type="button"
          className="bg-sage min-h-12 w-full rounded-full px-6 font-semibold text-white"
        >
          Sign in
        </button>
      </div>
    ),
  },
} satisfies Meta<typeof AuthCard>;

export default meta;
type Story = StoryObj<typeof meta>;

export const Default: Story = {};
