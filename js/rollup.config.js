var typescript = require("@rollup/plugin-typescript");
var commonjs = require("@rollup/plugin-commonjs");
var replace = require("@rollup/plugin-replace");
var strip = require("@rollup/plugin-strip");
var terser = require("@rollup/plugin-terser");

function bundle(src, name, { js = false, minify = false } = {}) {
  return {
    input: `./src/${src}.${js ? "js" : "ts"}`,
    output: [
      {
        file: `../modules/node_modules/${src}.es6${minify ? ".min" : ""}.js`,
        format: "umd",
        name: name,
        sourcemap: true,
      },
    ],
    plugins: [
      commonjs(),
      typescript({
        tsconfig: "./tsconfig.json",
        declaration: false,
        declarationMap: false,
      }),
      replace({
        include: "src/**/*.ts",
        preventAssignment: true,
        values: {
          DEBUG: false,
        },
      }),
      strip({
        include: "src/**/*.ts",
        functions: ["assert"],
        sourceMap: true,
      }),
      minify ? terser() : undefined,
    ].filter(Boolean),
  };
}

module.exports = [
  bundle("url-search-params/polyfill", "URLSearchParamsPolyfill", {
    js: true,
    minify: true,
  }),
  bundle("react-native-fetch/polyfill", "StreamFetchPolyfill", {
    minify: true,
  }),
  bundle("web-streams/polyfill", "WebStreamsPolyfill", {
    minify: true,
  }),
  bundle("text-encoding/FastestTextEncoderPolyfill", "TextEncoder", {
    js: true,
    minify: true,
  }),
  bundle("text-encoding/FastestTextDecoderPolyfill", "TextDecoder", {
    js: true,
    minify: true,
  }),
];
