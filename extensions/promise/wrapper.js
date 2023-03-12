"use strict";
const __runtimeResolverFuncWithArg = (fn, // JavaScript native function, usually the handler in the script
arg, // argument to the said handler as native object
resolve, // callback to Go when the Promise resolves
reject // callback to Go when the Promise rejects
) => {
    Promise.resolve(fn(arg)).then(resolve).catch(reject);
};
const __runtimeResolverFuncWithSpread = (fn, // JavaScript native function, usually the handler in the script
arg, // argument to the said handler as native object
resolve, // callback to Go when the Promise resolves
reject // callback to Go when the Promise rejects
) => {
    Promise.resolve(fn(arg))
        .then((r) => {
        resolve(...r);
    })
        .catch(reject);
};
