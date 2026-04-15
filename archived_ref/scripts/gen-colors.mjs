// Generate the --c-* color tonal palettes from a seed hex.
// Run via: node scripts/gen-colors.mjs <hex>
// Default seed is the Stackific brand: #1447E6
//
// Output is printed to stdout — paste it into the COLOR section of
// src/styles/tokens.css. Manual paste is intentional so the rest of
// tokens.css can stay hand-organized.

import { CorePalette, argbFromHex, hexFromArgb } from "@material/material-color-utilities";

const seed = process.argv[2] ?? "#1447E6";
const argb = argbFromHex(seed);
const palette = CorePalette.of(argb);

const TONES = [0, 5, 10, 15, 20, 25, 30, 35, 40, 50, 60, 70, 80, 90, 95, 98, 99, 100];
const KEYS = {
  primary: "a1",
  secondary: "a2",
  tertiary: "a3",
  neutral: "n1",
  "neutral-variant": "n2",
  error: "error",
};

console.log(`/* Generated from seed ${seed} via @material/material-color-utilities */`);
console.log("");
for (const [name, key] of Object.entries(KEYS)) {
  console.log(`  /* ${name[0].toUpperCase() + name.slice(1)} palette */`);
  for (const t of TONES) {
    const hex = hexFromArgb(palette[key].tone(t));
    const pad = String(t).padStart(3, " ");
    console.log(`  --c-${name}-${t}:${" ".repeat(4 - String(t).length)}${hex};`);
  }
  console.log("");
}
