import "@fontsource-variable/geist/wght.css";
import "beercss";
import "material-dynamic-colors";
import { mount } from "svelte";
import App from "./App.svelte";
import "./app.scss";
import { initTheme } from "./lib/theme.js";

await initTheme();

mount(App, { target: document.getElementById("app") });
