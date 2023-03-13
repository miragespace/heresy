const __runtimeResolverResult = <T>(
  arg: T | Promise<T>, // argument as native object
  resolve: (result: T) => void, // callback to Go when the Promise resolves
  reject: (e: unknown) => void // callback to Go when the Promise rejects
) => {
  Promise.resolve(arg).then(resolve).catch(reject);
};

const __runtimeResolverFuncWithArg = <T>(
  fn: (arg: T) => T | Promise<T>, // JavaScript native function, usually the handler in the script
  arg: T, // argument to the said handler as native object
  resolve: (result: T) => void, // callback to Go when the Promise resolves
  reject: (e: unknown) => void // callback to Go when the Promise rejects
) => {
  __runtimeResolverResult(fn(arg), resolve, reject);
};

const __runtimeResolverFuncWithSpread = <T>(
  fn: (arg: T) => T[] | Promise<T[]>, // JavaScript native function, usually is the __RuntimeResponseHelper
  arg: any, // argument to the said handler as native object
  resolve: (...result: T[]) => void, // callback to Go when the Promise resolves
  reject: (e: unknown) => void // callback to Go when the Promise rejects
) => {
  Promise.resolve(fn(arg))
    .then((r) => {
      resolve(...r);
    })
    .catch(reject);
};
