"use strict";
const __runtimeResolver = (fn, arg, resolve, reject) => {
    Promise.resolve(fn(arg)).then(resolve).catch(reject);
};
