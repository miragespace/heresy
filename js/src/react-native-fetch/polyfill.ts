import { globals } from "../utils";
import Headers from "./Headers";
import Request from "./Request";
import Response from "./Response";

const m = {
  Headers,
  Request,
  Response,
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
