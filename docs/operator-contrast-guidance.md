# Operator Guidance: Colour Contrast for Custom Accent Colours

Cuttlegate allows operators to customise the application's accent colour by overriding the `--color-accent` CSS custom property in a runtime configuration stylesheet. This variable controls:

- Primary button backgrounds
- Focus ring outlines on interactive elements (buttons, links, inputs)
- Any other UI elements styled with `[var(--color-accent)]`

## WCAG 2.1 AA Contrast Requirements

To meet WCAG 2.1 Level AA, your chosen accent colour must satisfy the following contrast ratios:

| Usage | Minimum contrast ratio | Against |
|---|---|---|
| Text on accent background (e.g. white text on primary button) | **4.5:1** | The accent colour itself |
| Large text (≥18pt / ≥14pt bold) on accent background | **3:1** | The accent colour itself |
| Focus ring outlines | **3:1** | The adjacent background colour |

The default accent colour (`#2563eb`, Tailwind `blue-600`) passes all of these checks against both white (`#ffffff`) and the application's light grey background (`#f9fafb`).

## Checking Your Colour

Use one of these tools to verify contrast ratios before deploying a custom accent colour:

- [WebAIM Contrast Checker](https://webaim.org/resources/contrastchecker/)
- [Accessible Colors](https://accessible-colors.com/)
- Browser DevTools accessibility panel (Chrome, Firefox, Safari)

## How to Override

Add a stylesheet that the server serves and inject it via the `BRANDING_CSS_URL` environment variable (see issue #194 for the full branding configuration reference). Minimal example:

```css
:root {
  --color-accent: #0f766e; /* Tailwind teal-700 — passes AA against white */
}
```

## Operator Responsibility

Cuttlegate enforces contrast for the default theme. When you supply a custom `--color-accent` value, **you are responsible** for verifying that it meets the WCAG 2.1 AA ratios listed above. The application cannot programmatically validate arbitrary runtime CSS values.
