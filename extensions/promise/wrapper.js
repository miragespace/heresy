"use strict";
const __runtimeResolverResult = (arg, // argument as native object
resolve, // callback to Go when the Promise resolves
reject // callback to Go when the Promise rejects
) => {
    Promise.resolve(arg).then(resolve).catch(reject);
};
const __runtimeResolverFuncWithArg = (fn, // JavaScript native function, usually the handler in the script
arg, // argument to the said handler as native object
resolve, // callback to Go when the Promise resolves
reject // callback to Go when the Promise rejects
) => {
    __runtimeResolverResult(fn(arg), resolve, reject);
};
