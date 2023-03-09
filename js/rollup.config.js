var typescript = require("@rollup/plugin-typescript");
var terser = require("@rollup/plugin-terser");

function bundle(src, name, { minify = false } = {}) {
  return {
    input: `./src/${src}.ts`,
    output: [
      {
        file: `../modules/node_modules/${src}.es6${minify ? ".min" : ""}.js`,
        format: "umd",
        name: name,
        sourcemap: true,
      },
    ],
    plugins: [
      typescript({
        tsconfig: "./tsconfig.json",
        declaration: false,
        declarationMap: false,
      }),
      minify ? terser() : undefined,
    ].filter(Boolean),
  };
}

module.exports = [
  bundle("fetch/polyfill", "FetchPolyfill"),
  bundle("fetch/polyfill", "FetchPolyfill", { minify: true }),
];
