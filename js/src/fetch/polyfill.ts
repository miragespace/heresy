import Headers from "./headers";
import { globals } from "../utils";

export { Headers };

const m = {
  Headers,
};

// Add classes to global scope
if (typeof globals !== "undefined") {
  for (const prop in m) {
    if (Object.prototype.hasOwnProperty.call(m, prop)) {
      Object.defineProperty(globals, prop, {
        value: m[prop as keyof typeof m],
        writable: true,
        configurable: true,
      });
    }
  }
}
