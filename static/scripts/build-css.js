#!/usr/bin/env node
// Build pipeline: Lightning CSS → PurgeCSS → concatenate with BeerCSS.
//
// 1. Lightning CSS bundles + minifies custom CSS → css/dist/custom.css
// 2. PurgeCSS strips unused classes from custom.css (scans templates)
// 3. Rewrite BeerCSS font paths from relative to /fonts/
// 4. Concatenate beer.min.css + purged custom.css → css/dist/app.css

const { execSync } = require("child_process");
const fs = require("fs");
const path = require("path");

const root = path.resolve(__dirname, "..");

// Step 1: Bundle + minify custom CSS.
execSync(
  "./node_modules/.bin/lightningcss --bundle --minify --targets '>= 0.25%' css/src/app.css -o css/dist/custom.css",
  { cwd: root, stdio: "inherit" },
);

// Step 2: PurgeCSS on custom CSS only.
execSync("./node_modules/.bin/purgecss --config purgecss.config.cjs", {
  cwd: root,
  stdio: "inherit",
});

// Step 3: Read BeerCSS and fix font paths.
let beer = fs.readFileSync(path.join(root, "vendor/beer.min.css"), "utf8");
beer = beer.replaceAll(
  "url(material-symbols-",
  "url(/fonts/material-symbols-",
);

// Step 4: Read purged custom CSS and concatenate.
const custom = fs.readFileSync(path.join(root, "css/dist/custom.css"), "utf8");
fs.writeFileSync(path.join(root, "css/dist/app.css"), beer + "\n" + custom);

// Clean up intermediate file.
fs.unlinkSync(path.join(root, "css/dist/custom.css"));

const size = fs.statSync(path.join(root, "css/dist/app.css")).size;
console.log(`css/dist/app.css: ${(size / 1024).toFixed(1)} KB`);
