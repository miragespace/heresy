"use strict";
const __runtimeResolverResult = async (arg, // argument as native object
resolve, // callback to Go when the Promise resolves
reject // callback to Go when the Promise rejects
) => {
    try {
        const r = await arg;
        resolve(r);
    }
    catch (e) {
        reject(e);
    }
};
const __runtimeResolverFuncWithArg = (fn, // JavaScript native function, usually the handler in the script
arg, // argument to the said handler as native object
resolve, // callback to Go when the Promise resolves
reject // callback to Go when the Promise rejects
) => {
    __runtimeResolverResult(fn(arg), resolve, reject);
};
