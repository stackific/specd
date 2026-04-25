#!/usr/bin/env node
// Build pipeline: Sass → Lightning CSS → PurgeCSS → concatenate with BeerCSS + fonts.
//
// 1. Sass compiles SCSS → css/dist/custom.css
// 2. Lightning CSS bundles + minifies custom.css (in place)
// 3. PurgeCSS strips unused classes from custom.css (scans templates)
// 4. Rewrite BeerCSS font paths from relative to /fonts/
// 5. Read @fontsource-variable/geist CSS and rewrite font paths
// 6. Copy Geist woff2 files to fonts/
// 7. Concatenate beer.min.css + font CSS + purged custom.css → css/dist/app.css

const { execSync } = require("child_process");
const fs = require("fs");
const path = require("path");

const root = path.resolve(__dirname, "..");

// Step 1: Compile SCSS → CSS.
execSync(
  "./node_modules/.bin/sass --no-source-map --style=expanded css/src/app.scss css/dist/custom.css",
  { cwd: root, stdio: "inherit" },
);

// Step 2: Bundle + minify compiled CSS.
execSync(
  "./node_modules/.bin/lightningcss --bundle --minify --targets '>= 0.25%' css/dist/custom.css -o css/dist/custom.css",
  { cwd: root, stdio: "inherit" },
);

// Step 3: PurgeCSS on custom CSS only.
execSync("./node_modules/.bin/purgecss --config purgecss.config.cjs", {
  cwd: root,
  stdio: "inherit",
});

// Step 4: Read BeerCSS and fix font paths.
let beer = fs.readFileSync(path.join(root, "vendor/beer.min.css"), "utf8");
beer = beer.replaceAll(
  "url(material-symbols-",
  "url(/fonts/material-symbols-",
);

// Step 5: Read Geist font CSS and rewrite paths to /fonts/.
const fontsrcDir = path.join(
  root,
  "node_modules/@fontsource-variable/geist",
);
let fontCss = fs.readFileSync(path.join(fontsrcDir, "index.css"), "utf8");
fontCss = fontCss.replaceAll("url(./files/", "url(/fonts/");

// Step 6: Copy Geist woff2 files to fonts/.
const fontsDir = path.join(root, "fonts");
const srcFiles = path.join(fontsrcDir, "files");
for (const f of fs.readdirSync(srcFiles)) {
  if (f.endsWith(".woff2")) {
    fs.copyFileSync(path.join(srcFiles, f), path.join(fontsDir, f));
  }
}

// Step 7: Read purged custom CSS and concatenate.
const custom = fs.readFileSync(path.join(root, "css/dist/custom.css"), "utf8");
fs.writeFileSync(
  path.join(root, "css/dist/app.css"),
  beer + "\n" + fontCss + "\n" + custom,
);

// Clean up intermediate file.
fs.unlinkSync(path.join(root, "css/dist/custom.css"));

const size = fs.statSync(path.join(root, "css/dist/app.css")).size;
console.log(`css/dist/app.css: ${(size / 1024).toFixed(1)} KB`);
