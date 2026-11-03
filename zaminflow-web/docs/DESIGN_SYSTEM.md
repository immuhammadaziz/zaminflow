# Approved design system

## Direction

“Grounded modern trade”: preserve the live site's warm ivory surfaces, espresso wordmark/headlines, sage accents and editorial restraint, while making signed-in workspaces feel precise and data-forward. Avoid template gradients, cartoon dairy imagery and invented live statistics.

## Color tokens and measured contrast

Ratios were computed using WCAG relative luminance against `#F7F5EF` and `#FFFFFF`. Normal text uses only combinations meeting 4.5:1; status is never conveyed by color alone.

| Token     | Value     | Use                          | Contrast on ivory / white                              |
| --------- | --------- | ---------------------------- | ------------------------------------------------------ |
| `canvas`  | `#F7F5EF` | Warm page background         | —                                                      |
| `surface` | `#FFFFFF` | Cards/forms                  | —                                                      |
| `ink`     | `#2B211B` | Headings/body                | 14.42:1 / 15.72:1                                      |
| `sage`    | `#2F5D46` | Primary actions/focus accent | 6.94:1 / 7.57:1                                        |
| `muted`   | `#5E6259` | Secondary labels             | 5.72:1 / 6.24:1                                        |
| `gold`    | `#87520F` | Warning text/icon            | 5.94:1 / 6.47:1                                        |
| `danger`  | `#8B2F24` | Error/destructive text/icon  | 7.61:1 / 8.30:1                                        |
| `info`    | `#315B76` | Informational text/icon      | 6.67:1 / 7.27:1                                        |
| `border`  | `#D9D5C9` | Non-text boundaries          | Verify against adjacent control/background in final UI |

Status chips combine a distinct icon, explicit localized status text and contrasting border/fill. Charts use text/table alternatives and patterns/markers in addition to color.

## Type, rhythm and motion

- Two families maximum: existing logo asset for the wordmark; editorial serif for large marketing headings; system sans stack for controls and app UI until a properly licensed, self-hosted Latin/Cyrillic font is selected.
- Verify `oʻ` U+02BB and `gʻ` U+02BC plus Cyrillic before release. Use tabular numerals for UZS and liters.
- 8 px spacing grid; 4 px optical half-step only for icon alignment. Type steps: 12, 14, 16, 20, 24, 32, 40, 56 px with responsive clamps for hero headings.
- Radius: 8 px controls, 12 px cards, 20 px marketing panels. Subtle 1 px borders and restrained shadows.
- Transitions 150–250 ms; disable nonessential movement for `prefers-reduced-motion`.
- At least 44 px interactive targets. Forms stay single column on mobile; tables become cards; farmer navigation becomes a bottom bar.

## Component set

Buttons and loading states; localized form fields; password reveal/help; error/field summaries; role/status badges; cards; skeletons; empty/offline/retry states; confirmation dialog; accessible pagination and sortable table; mobile data cards; date/period filters; chart with data table; toast/inline feedback; breadcrumb; responsive app shell; mobile bottom navigation; locale menu.

## Theme

Light theme only for the first release, matching the current site and avoiding a second contrast/audit matrix. Revisit dark mode only after explicit product approval.

## Static high-fidelity previews

These are design artifacts, not application code. Each SVG places a desktop and mobile composition side by side:

- [Landing](./previews/landing.svg)
- [Sign-up](./previews/signup.svg)
- [Farmer dashboard](./previews/farmer-dashboard.svg)
- [Admin review queue](./previews/admin-review.svg)

Dashboard metrics use loading placeholders and an honest empty state; admin applicant rows are neutral placeholders. No fabricated business or personal data is represented.
