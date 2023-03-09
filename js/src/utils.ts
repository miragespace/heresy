/// <reference lib="dom" />

function getGlobals() {
  if (typeof self !== "undefined") {
    return self;
  } else if (typeof window !== "undefined") {
    return window;
  } else if (typeof globalThis !== "undefined") {
    return globalThis;
  }
  return undefined;
}

export const globals = getGlobals();
