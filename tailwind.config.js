/** @type {import('tailwindcss').Config} */
// const colors = require('tailwindcss/colors');
// const plugin = require("tailwindcss/plugin");

module.exports = {
  darkMode: "selector",
  // important: true,
  content: ["view/**/*.templ", "view/static/**/*.html"],
  safelist: [
    {
      pattern: /^(?!(?:scroll|bottom)$)m\w?-/,
      variants: ["sm", "md", "lg", "xl", "2xl"],
    },
    {
      pattern: /^(?!(?:scroll|bottom)$)p\w?-/,
      variants: ["sm", "md", "lg", "xl", "2xl"],
    },
  ],
};
